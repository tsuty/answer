package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"
)

var logger *Logger

func main() {
	var opts struct {
		Help bool `short:"h" long:"help" description:"Show this help message"`
		// log
		LogFile  string `long:"log" description:"The log file path (default stdout)"`
		Loglevel string `long:"level" description:"The log level" choice:"debug" choice:"info" choice:"notice" choice:"warn" choice:"error" default:"debug"`
		// dns
		Host         string `long:"host" description:"The host name" default:"127.0.0.1"`
		Port         string `long:"port" description:"The port number (TCP and UDP)" default:"53"`
		ReadTimeout  string `long:"read-timeout" description:"The read timeout" default:"5s" hidden:"1"`
		WriteTimeout string `long:"write-timeout" description:"The write timeout" default:"5s" hidden:"1"`
	}

	parser := flags.NewParser(&opts, flags.None)
	parser.Name = "answer"
	parser.LongDescription = `tiny DNS proxy`

	_, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments\n%v\n%s\n\n", os.Args[1:], err.Error())
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
		return
	}

	if opts.Help {
		parser.WriteHelp(os.Stdout)
		return
	}

	logger, err = NewLogger(opts.LogFile, opts.Loglevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logging Error\n%s", err.Error())
		os.Exit(1)
		return
	}
	logger.Info("boot up ...")

	servers, err := NewServers(opts.Host,
		opts.Port,
		opts.ReadTimeout,
		opts.WriteTimeout)
	if err != nil {
		logger.Error("failed to setup server %s", err.Error())
		logger.Shutdown()
		os.Exit(1)
		return
	}

	logger.Info("start servers ...")
	if err := servers.Start(); err != nil {
		logger.Error("failed to start server %s", err.Error())
		logger.Shutdown()
		os.Exit(1)
		return
	}

	sig := make(chan os.Signal)
	signal.Notify(sig,
		os.Interrupt,
		syscall.SIGKILL,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

loop:
	for {
		select {
		case <-sig:
			logger.Info("shutdown servers ...")
			break loop
		}
	}

	logger.Shutdown()
}
