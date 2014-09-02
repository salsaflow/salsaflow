package log

import (
	"fmt"
	"os"
	"sync/atomic"
)

type (
	Level  uint32
	Logger bool
)

const (
	Trace = iota
	Debug
	Verbose
	Info
	Off
)

var v Level = Info

func SetV(level Level) {
	atomic.StoreUint32((*uint32)(&v), uint32(level))
}

func V(level Level) Logger {
	if atomic.LoadUint32((*uint32)(&v)) > uint32(level) {
		return Logger(false)
	}
	return Logger(true)
}

func (l Logger) log(v ...interface{}) {
	if l {
		fmt.Fprint(os.Stderr, v...)
	}
}

func (l Logger) logf(format string, v ...interface{}) {
	if l {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}

func (l Logger) logln(v ...interface{}) {
	if l {
		fmt.Fprintln(os.Stderr, v...)
	}
}

func (l Logger) Run(msg string) {
	l.logf("[RUN]  %v\n", msg)
}

func (l Logger) Skip(msg string) {
	l.logf("[SKIP] %v\n", msg)
}

func (l Logger) Go(msg string) {
	l.logf("[GO]   %v\n", msg)
}

func (l Logger) Ok(msg string) {
	l.logf("[OK]   %v\n", msg)
}

func (l Logger) Fail(msg string) {
	l.logf("[FAIL] %v\n", msg)
}

func (l Logger) Print(v ...interface{}) {
	l.log(v...)
}

func (l Logger) Printf(format string, v ...interface{}) {
	l.logf(format, v...)
}

func (l Logger) Println(v ...interface{}) {
	l.logln(v...)
}

func (l Logger) Fatal(v ...interface{}) {
	fmt.Fprint(os.Stderr, v...)
	os.Exit(1)
}

func (l Logger) Fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	os.Exit(1)
}

func (l Logger) Fatalln(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func Run(msg string) {
	V(Info).Run(msg)
}

func Skip(msg string) {
	V(Info).Run(msg)
}

func Go(msg string) {
	V(Info).Go(msg)
}

func Ok(msg string) {
	V(Info).Ok(msg)
}

func Fail(msg string) {
	V(Info).Fail(msg)
}

func Print(v ...interface{}) {
	V(Info).Print(v...)
}

func Printf(format string, v ...interface{}) {
	V(Info).Printf(format, v...)
}

func Println(v ...interface{}) {
	V(Info).Println(v...)
}

func Fatal(v ...interface{}) {
	V(Info).Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	V(Info).Fatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	V(Info).Fatalln(v...)
}
