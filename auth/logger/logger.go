package logger

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
)

type Logger struct {
	errChan  chan error
	warnChan chan string
	infoChan chan string

	once sync.Once
}

type LoggerInterface interface {
	StartListener()
	OpenLogFile(logFileName string, flag int, perm os.FileMode) *os.File
	Error(err error)
	ErrorF(err string, args ...any)
	ErrorStr(err string, args ...any)
	Warn(warning string)
	Info(info string)
	InfoF(info string, args ...any)
	WarnF(warning string, args ...any)
}

func NewLogger() LoggerInterface {
	return &Logger{
		errChan:  make(chan error, 100),
		warnChan: make(chan string, 100),
		infoChan: make(chan string, 100),
	}
}

// StartListener This spins up a goroutine that listens on error channel and writes to logs.
// This design choice was made to avoid blocking the main execution flow with overhead of writing to a file.
// Writing to a file could include system calls, mutexes(to avoid race conditions).
// So tens of hundreds of goroutines waiting for a single synchronization technique on logging is not an ideal choice.
// This avoids it by having a separate goroutine listening to a non-blocking buffered channel.
// The rest of the goroutines will just write to a channel and go on with their lives.
func (l *Logger) StartListener() {
	// Make sure listener is initialized only once
	l.once.Do(func() {
		// TODO: Potentially need to configure stout logger for dev env
		errLogFile := l.OpenLogFile("logger/errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		warnLogFile := l.OpenLogFile("logger/warnings.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		infoLogFile := l.OpenLogFile("logger/info.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		// Create logger instances
		errLogger := slog.New(slog.NewTextHandler(errLogFile, nil))
		warnLogger := slog.New(slog.NewTextHandler(warnLogFile, nil))
		infoLogger := slog.New(slog.NewTextHandler(infoLogFile, nil))

		// Listen to ErrChan
		go func(el, wl, il *slog.Logger) {
			for {
				select {
				case err := <-l.errChan:
					// Error types can be nil, check for it before logging
					if err != nil {
						el.Error(err.Error())
						fmt.Println(err.Error())
					}
				case warning := <-l.warnChan:
					wl.Warn(warning)
					fmt.Println(warning)
				case info := <-l.infoChan:
					il.Info(info)
					fmt.Println(info)
				}
			}
		}(errLogger, warnLogger, infoLogger)
	})
}

func (l *Logger) OpenLogFile(logFileName string, flag int, perm os.FileMode) *os.File {
	logFile, err := os.OpenFile(logFileName, flag, perm)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not open log file: %s, %v", logFileName, err))
	}

	return logFile
}

func (l *Logger) Error(err error) {
	l.errChan <- err
}

func (l *Logger) ErrorF(err string, args ...any) {
	l.errChan <- fmt.Errorf(err, args...)
}

func (l *Logger) ErrorStr(err string, args ...any) {
	l.errChan <- fmt.Errorf(err, args...)
}

func (l *Logger) Warn(warning string) {
	l.warnChan <- warning
}

func (l *Logger) WarnF(warning string, args ...any) {
	l.warnChan <- fmt.Sprintf(warning, args...)
}

func (l *Logger) Info(info string) {
	l.infoChan <- info
}

func (l *Logger) InfoF(info string, args ...any) {
	l.infoChan <- fmt.Sprintf(info, args...)
}
