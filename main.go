package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	flags "github.com/jessevdk/go-flags"
	"github.com/kazeburo/mackerel-plugin-axslog/axslog"
	"github.com/kazeburo/mackerel-plugin-axslog/posreader"
	"github.com/kazeburo/mackerel-plugin-postfix-log/postfixlog"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Version by Makefile
var Version string

// MaxReadSize : Maximum size for read
var MaxReadSize int64 = 500 * 1000 * 1000

type cmdOpts struct {
	LogFile       string `long:"logfile" default:"/var/log/maillog" description:"path to nginx ltsv logfile" required:"true"`
	PosFilePrefix string `long:"posfile-prefix" default:"maillog" description:"prefix added position file"`
	Version       bool   `short:"v" long:"version" description:"Show version"`
}

var logFilter = []byte(" postfix/smtp[")

func parseLog(bs *bufio.Scanner, logger *zap.Logger) (*postfixlog.Stats, error) {
	for bs.Scan() {
		b := bs.Bytes()
		if bytes.Index(b, logFilter) < 0 {
			continue
		}
		s, err := postfixlog.Parse(b)
		if err != nil {
			logger.Warn("Failed to convert status. continue", zap.Error(err))
			continue
		}
		return s, nil

	}
	if bs.Err() != nil {
		return nil, bs.Err()
	}
	return nil, io.EOF
}

func parseFile(logFile string, lastPos int64, posFile string, bin *postfixlog.StatsBin, logger *zap.Logger) error {
	stat, err := os.Stat(logFile)
	if err != nil {
		return errors.Wrap(err, "failed to stat log file")
	}

	fstat, err := axslog.FileStat(stat)
	if err != nil {
		return errors.Wrap(err, "failed to inode of log file")
	}

	logger.Info("Analysis start",
		zap.String("logFile", logFile),
		zap.Int64("lastPos", lastPos),
		zap.Int64("Size", stat.Size()),
	)

	if lastPos == 0 && stat.Size() > MaxReadSize {
		// first time and big logile
		lastPos = stat.Size()
	}

	if stat.Size()-lastPos > MaxReadSize {
		// big delay
		lastPos = stat.Size()
	}

	f, err := os.Open(logFile)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	defer f.Close()
	fpr, err := posreader.New(f, lastPos)
	if err != nil {
		return errors.Wrap(err, "failed to seek log file")
	}

	total := 0
	bs := bufio.NewScanner(fpr)
	for {
		s, e := parseLog(bs, logger)
		if e == io.EOF {
			break
		}
		if e != nil {
			return errors.Wrap(e, "Something wrong in parse log")
		}
		bin.Append(s)
		total++
	}

	logger.Info("Analysis completed",
		zap.String("logFile", logFile),
		zap.Int64("startPos", lastPos),
		zap.Int64("endPos", fpr.Pos),
		zap.Int("Rows", total),
	)
	// update postion
	if posFile != "" {
		err = axslog.WritePos(posFile, fpr.Pos, fstat)
		if err != nil {
			return errors.Wrap(err, "failed to update pos file")
		}
	}
	return nil
}

func getStats(opts cmdOpts, logger *zap.Logger) error {
	lastPos := int64(0)
	lastFstat := &axslog.FStat{}
	tmpDir := os.TempDir()
	curUser, _ := user.Current()
	uid := "0"
	if curUser != nil {
		uid = curUser.Uid
	}
	posFile := filepath.Join(tmpDir, fmt.Sprintf("%s-postfixlog-%s-v1", opts.PosFilePrefix, uid))
	duration := float64(0)
	bin := postfixlog.NewStatsBin()

	if axslog.FileExists(posFile) {
		l, d, f, err := axslog.ReadPos(posFile)
		if err != nil {
			return errors.Wrap(err, "failed to load pos file")
		}
		lastPos = l
		duration = d
		lastFstat = f
	}

	stat, err := os.Stat(opts.LogFile)
	if err != nil {
		return errors.Wrap(err, "failed to stat log file")
	}
	fstat, err := axslog.FileStat(stat)
	if err != nil {
		return errors.Wrap(err, "failed to get inode from log file")
	}
	if fstat.IsNotRotated(lastFstat) {
		err := parseFile(
			opts.LogFile,
			lastPos,
			posFile,
			bin,
			logger,
		)
		if err != nil {
			return err
		}
	} else {
		// rotate!!
		logger.Info("Detect Rotate")
		lastFile, err := axslog.SearchFileByInode(filepath.Dir(opts.LogFile), lastFstat)
		if err != nil {
			logger.Warn("Could not search previous file",
				zap.Error(err),
			)
			// new file
			err := parseFile(
				opts.LogFile,
				0, // lastPos
				posFile,
				bin,
				logger,
			)
			if err != nil {
				return err
			}
		} else {
			// new file
			err := parseFile(
				opts.LogFile,
				0, // lastPos
				posFile,
				bin,
				logger,
			)
			if err != nil {
				return err
			}
			// previous file
			err = parseFile(
				lastFile,
				lastPos,
				"", // no update posfile
				bin,
				logger,
			)
			if err != nil {
				logger.Warn("Could not parse previous file",
					zap.Error(err),
				)
			}
		}
	}

	bin.Display(duration)

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
