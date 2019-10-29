package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	flags "github.com/jessevdk/go-flags"
	"github.com/kazeburo/followparser"
	"github.com/kazeburo/mackerel-plugin-postfix-log/postfixlog"
	"go.uber.org/zap"
)

// Version by Makefile
var Version string

type cmdOpts struct {
	LogFile       string `long:"logfile" default:"/var/log/maillog" description:"path to nginx ltsv logfile" required:"true"`
	PosFilePrefix string `long:"posfile-prefix" default:"maillog" description:"prefix added position file"`
	Version       bool   `short:"v" long:"version" description:"Show version"`
}

var logFilter = []byte(" postfix/smtp[")

type parser struct {
	bin *postfixlog.StatsBin
}

func (p *parser) Parse(b []byte) error {
	if bytes.Index(b, logFilter) < 0 {
		return nil
	}
	s, err := postfixlog.Parse(b)
	if err != nil {
		return err
	}
	p.bin.Append(s)
	return nil
}

func (p *parser) Finish(duration float64) {
	p.bin.Display(duration)
}

func getStats(opts cmdOpts, logger *zap.Logger) error {
	bin := postfixlog.NewStatsBin()
	p := &parser{bin}
	err := followparser.Parse(opts.PosFilePrefix, opts.LogFile, p, logger)
	if err != nil {
		return err
	}
	return nil

}

func printVersion() {
	fmt.Printf(`%s %s
Compiler: %s %s
`,
		os.Args[0],
		Version,
		runtime.Compiler,
		runtime.Version())
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opts := cmdOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	_, err := psr.Parse()
	if err != nil {
		return 1
	}
	if opts.Version {
		printVersion()
		return 0
	}

	logger, _ := zap.NewProduction()
	err = getStats(opts, logger)
	if err != nil {
		logger.Error("getStats", zap.Error(err))
		return 1
	}
	return 0
}
