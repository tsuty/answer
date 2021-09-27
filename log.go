package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	UnknownLogLevel LogLevel = iota
	Debug
	Info
	Notice
	Warn
	Error
)

var logLevels = map[string]LogLevel{
	"debug":  Debug,
	"info":   Info,
	"notice": Notice,
	"warn":   Warn,
	"error":  Error,
}

var logLevelLabels = map[LogLevel]string{
	Debug:  "debug",
	Info:   "info",
	Notice: "notice",
	Warn:   "warn",
	Error:  "error",
}

const LogBuffer = 64

func (l LogLevel) String() string {
	return logLevelLabels[l]
}

const (
	LogOutputStdout = "stdout"
	LogOutputStdErr = "stderr"
)

type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (int, error) {
	return 0, nil
}

type Logger struct {
	level    LogLevel
	msg      chan string
	internal *log.Logger
	wg       *sync.WaitGroup
}

func NewLogger(file, level string) (*Logger, error) {
	lv, ok := logLevels[level]
	if !ok {
		return nil, fmt.Errorf("invalid log level %s", level)
	}

	var w io.Writer
	switch file {
	case "":
		w = os.Stdout
	default:
		var err error
		if w, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
			return nil, fmt.Errorf("failed to open file %s %w", file, err)
		}
	}

	l := &Logger{
		level:    lv,
		msg:      make(chan string, LogBuffer),
		internal: log.New(w, "", log.Ldate|log.Lmicroseconds),
		wg:       new(sync.WaitGroup),
	}

	go l.run()

	l.Info("start logging (level: %s)", level)

	return l, nil
}

func (l *Logger) Shutdown() {
	l.Wait()
}

func (l *Logger) Wait() {
	_, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	l.wg.Wait()
	return
}

func (l *Logger) run() {
	for {
		select {
		case msg := <-l.msg:
			l.write(msg)
		}
	}
}

func (l *Logger) write(msg string) {
	defer l.wg.Done()
	l.internal.Println(msg)
}

func (l *Logger) Print(level LogLevel, message string) {
	if l.level > level {
		return
	}

	msg := fmt.Sprintf("[%s] %s", logLevelLabels[level], message)
	l.wg.Add(1)
	l.msg <- msg
}

func (l *Logger) Debug(format string, a ...interface{}) {
	l.Print(Debug, fmt.Sprintf(format, a...))
}

func (l *Logger) Info(format string, a ...interface{}) {
	l.Print(Info, fmt.Sprintf(format, a...))
}

func (l *Logger) Notice(format string, a ...interface{}) {
	l.Print(Notice, fmt.Sprintf(format, a...))
}

func (l *Logger) Warn(format string, a ...interface{}) {
	l.Print(Warn, fmt.Sprintf(format, a...))
}

func (l *Logger) Error(format string, a ...interface{}) {
	l.Print(Error, fmt.Sprintf(format, a...))
}
