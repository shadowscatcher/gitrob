package common

import (
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
)

const (
	FATAL     = 5
	ERROR     = 4
	WARN      = 3
	IMPORTANT = 2
	INFO      = 1
	DEBUG     = 0
)

var LogColors = map[int]*color.Color{
	FATAL:     color.New(color.FgRed).Add(color.Bold),
	ERROR:     color.New(color.FgRed),
	WARN:      color.New(color.FgYellow),
	IMPORTANT: color.New(color.Bold),
	DEBUG:     color.New(color.FgCyan).Add(color.Faint),
}

type Logger struct {
	sync.Mutex

	debug  bool
	silent bool
}

func (l *Logger) SetSilent(s bool) {
	l.silent = s
}

func (l *Logger) SetDebug(d bool) {
	l.debug = d
}

func (l *Logger) Logf(level int, format string, args ...interface{}) {
	l.Lock()
	defer func() {
		if level == FATAL {
			os.Exit(1)
		}
		l.Unlock()
	}()

	if level == DEBUG && !l.debug {
		return
	} else if level < ERROR && l.silent {
		return
	}

	if c, ok := LogColors[level]; ok {
		_, _ = c.Printf(format, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logf(FATAL, format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(ERROR, format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logf(WARN, format, args...)
}

func (l *Logger) Importantf(format string, args ...interface{}) {
	l.Logf(IMPORTANT, format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logf(INFO, format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logf(DEBUG, format, args...)
}
