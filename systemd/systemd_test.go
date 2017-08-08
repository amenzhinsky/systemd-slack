package systemd

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	sd, err := New(
		WithStateFile(f.Name()),
		WithLogger(log.New(os.Stderr, "", log.LstdFlags)),
		WithInterval(100*time.Millisecond),
	)

	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cerr := sd.Close()
		if err == nil && cerr != nil {
			t.Fatal(cerr)
		}
	}()
}
