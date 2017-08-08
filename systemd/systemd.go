package systemd

import (
	"compress/gzip"
	"encoding/gob"
	"log"
	"os"
	"time"

	"github.com/coreos/go-systemd/dbus"
)

var (
	DefaultStateFile = "systemd.state"
	DefaultInterval  = 500 * time.Millisecond
)

// Option is a configuration value.
type Option func(sd *Systemd)

// WithStateFile sets path to the state file, if the file doesn't
// exist it's created automatically when systemd flushes its state.
func WithStateFile(path string) Option {
	return func(sd *Systemd) {
		sd.statePath = path
	}
}

// WithLogger sets logger, nil disables logging.
func WithLogger(l *log.Logger) Option {
	return func(sd *Systemd) {
		sd.logger = l
	}
}

// WithInterval sets systemd the interval between the ListUnits api call.
func WithInterval(d time.Duration) Option {
	return func(sd *Systemd) {
		sd.interval = d
	}
}

// New returns a systemd instance.
func New(opts ...Option) (*Systemd, error) {
	c, err := dbus.New()
	if err != nil {
		return nil, err
	}

	sd := &Systemd{
		conn:      c,
		state:     make(map[string]Unit),
		statePath: DefaultStateFile,
		interval:  DefaultInterval,
		logger:    log.New(os.Stdout, "[systemd] ", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(sd)
	}

	// load state
	if err = sd.load(); err != nil {
		return nil, err
	}
	return sd, nil
}

// Systemd is an units watcher.
type Systemd struct {
	conn      conn
	state     map[string]Unit
	statePath string
	logger    *log.Logger
	interval  time.Duration
	bootstrap bool
}

// conn is needed to mock systemd connection in tests
type conn interface {
	ListUnits() ([]dbus.UnitStatus, error)
	Close()
}

// Next
func (sd *Systemd) Next() ([]Unit, error) {
	first := true

	for {
		units, err := sd.conn.ListUnits()
		if err != nil {
			return nil, err
		}

		flush := false
		for _, s := range units {
			if unit, ok := sd.state[string(s.Path)]; ok && unit.isEqual(s) {
				continue
			}

			flush = true
			sd.state[string(s.Path)] = Unit{s}

			// don't report anything on the first run
			if sd.bootstrap && first {
				continue
			}

			// ActiveState
			//
			// active
			// inactive
			// activating
			// deactivating
			// failed

			// LoadState
			//
			// loaded
			// not-found

			// SubState
			//
			// running
			// start-pre
			// stop-sig*

			sd.logf("%s active=%s load=%s sub=%s", s.Name, s.ActiveState, s.LoadState, s.SubState)
		}

	Loop:
		for path, u := range sd.state {
			for _, s := range units {
				if string(s.Path) == path {
					continue Loop
				}
			}

			flush = true
			delete(sd.state, path)
			sd.logf("%s deleted", u.Name)
		}

		first = false
		if flush {
			if err = sd.store(); err != nil {
				return nil, err
			}
		}
		time.Sleep(sd.interval)
	}
}

// load loads state from the state file.
func (sd *Systemd) load() error {
	// bootstrap is enabled when the state file doesn't exist or it's empty.
	state, err := os.Lstat(sd.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			sd.bootstrap = true
			sd.logf("state file doesn't exist, enable bootstrap mode")
			return nil
		}
		return err
	}
	if state.Size() == 0 {
		return nil
	}

	f, err := os.OpenFile(sd.statePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer r.Close()

	return gob.NewDecoder(r).Decode(&sd.state)
}

// store flushes current state to the state file.
func (sd *Systemd) store() error {
	f, err := os.OpenFile(sd.statePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()

	return gob.NewEncoder(w).Encode(sd.state)
}

// logf logs a message, arguments are treated like fmt.Sprintf.
func (sd *Systemd) logf(s string, v ...interface{}) {
	if sd.logger != nil {
		sd.logger.Printf(s, v...)
	}
}

// Close closes dbus connection.
func (sd *Systemd) Close() error {
	sd.conn.Close()
	return nil
}

// Unit is a unit status object.
type Unit struct {
	dbus.UnitStatus
}

// isEqual compares the unit to a dbus.UnitStatus.
func (u *Unit) isEqual(u2 dbus.UnitStatus) bool {
	return u.UnitStatus == u2
}
