package logx

import (
	"fmt"
	"io"
	"log"
)

type Logger struct {
	l *log.Logger
}

func New(w io.Writer) *Logger {
	return &Logger{l: log.New(w, "", log.Ltime)}
}

// WithPrefix returns a logger that tags every line with a fixed label (e.g.
// "TRACKING" or "DEALS") placed after the timestamp, so concurrent scans are
// easy to tell apart. The underlying writer is shared with the parent.
func (l *Logger) WithPrefix(label string) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{l: log.New(l.l.Writer(), fmt.Sprintf("%-8s ", label), log.Ltime|log.Lmsgprefix)}
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l == nil {
		return
	}
	l.l.Printf(format, args...)
}
