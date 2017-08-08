package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/amenzhinsky/consul-slack/slack"
	"github.com/amenzhinsky/systemd-slack/systemd"
)

var (
	channelFlag  = "systemd-state"
	usernameFlag = "systemd"
	iconURLFlag  = "https://emoji.slack-edge.com/T043Q7UHW/garold/269d90c3a5ffe40f.png"

	stateFileFlag = systemd.DefaultStateFile
	intervalFlag  = systemd.DefaultInterval
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s SLACK_WEEBHOOK_URL\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&channelFlag, "slack-channel", channelFlag, "slack channel name")
	flag.StringVar(&usernameFlag, "slack-username", usernameFlag, "slack username")
	flag.StringVar(&iconURLFlag, "slack-icon-url", iconURLFlag, "slack avatar url")
	flag.StringVar(&stateFileFlag, "state-file", stateFileFlag, "path to the state file")
	flag.DurationVar(&intervalFlag, "interval", intervalFlag, "status polling interval")
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

// start ensures that all defers are executed before the process exits.
func start() error {
	s, err := slack.New(flag.Arg(0),
		slack.WithChannel(channelFlag),
		slack.WithUsername(usernameFlag),
		slack.WithIconURL(iconURLFlag),
	)
	if err != nil {
		return err
	}

	_ = s // TODO: use it

	sd, err := systemd.New(
		systemd.WithStateFile(stateFileFlag),
		systemd.WithInterval(intervalFlag),
	)
	if err != nil {
		return err
	}
	defer sd.Close()

	for {
		changes, err := sd.Next()
		if err != nil {
			return err
		}

		for _, c := range changes {
			fmt.Printf("--> %#v\n", c)
		}
	}
}
