package app

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	TraceBLEPath string
}

func ParseOptions(args []string) (Options, error) {
	var opts Options
	fs := flag.NewFlagSet("timeflip-desktop", flag.ContinueOnError)
	fs.StringVar(&opts.TraceBLEPath, "trace-ble", "", "write raw BLE operation trace to PATH, or '-' for stderr; trace includes password bytes")
	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	if fs.NArg() > 0 {
		return Options{}, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	return opts, nil
}

func Run() {
	opts, err := ParseOptions(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	RunWithOptions(opts)
}
