package log

import (
	"bytes"
	"errors"
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
	l.logf("[RUN]      %v\n", msg)
}

func (l Logger) Skip(msg string) {
	l.logf("[SKIP]     %v\n", msg)
}

func (l Logger) Go(msg string) {
	l.logf("[GO]       %v\n", msg)
}

func (l Logger) Ok(msg string) {
	l.logf("[OK]       %v\n", msg)
}

func (l Logger) Fail(msg string) {
	l.logf("[FAIL]     %v\n", msg)
}

func (l Logger) Rollback(msg string) {
	l.logf("[ROLLBACK] %v\n", msg)
}

func (l Logger) NewLine(msg string) {
	l.logf("           %v\n", msg)
}

func (l Logger) FailWithContext(msg string, stderr *bytes.Buffer) {
	if msg != "" {
		l.Fail(msg)
	}
	l.Stderr(stderr)
}

func (l Logger) Stderr(stderr *bytes.Buffer) {
	if stderr != nil && stderr.Len() != 0 {
		l.Println("<<<<< stderr")
		l.Print(stderr)
		l.Println(">>>>> stderr")
	}
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

func Rollback(msg string) {
	V(Info).Rollback(msg)
}

func NewLine(msg string) {
	V(Info).NewLine(msg)
}

func FailWithContext(msg string, stderr *bytes.Buffer) {
	V(Info).FailWithContext(msg, stderr)
}

func Stderr(stderr *bytes.Buffer) {
	V(Info).Stderr(stderr)
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

var levelToStringMap = map[Level]string{
	Trace:   "trace",
	Debug:   "debug",
	Verbose: "verbose",
	Info:    "info",
	Off:     "off",
}

func LevelToString(level Level) (string, bool) {
	v, ok := levelToStringMap[level]
	return v, ok
}

func MustLevelToString(level Level) string {
	v, ok := LevelToString(level)
	if !ok {
		panic(fmt.Errorf("invalid log level: %v", level))
	}
	return v
}

var stringToLevelMap = map[string]Level{
	"trace":   Trace,
	"debug":   Debug,
	"verbose": Verbose,
	"info":    Info,
	"off":     Off,
}

func StringToLevel(levelString string) (Level, bool) {
	v, ok := stringToLevelMap[levelString]
	return v, ok
}

func MustStringToLevel(levelString string) Level {
	level, ok := StringToLevel(levelString)
	if !ok {
		panic(errors.New("invalid log level string: " + levelString))
	}
	return level
}

func LevelStrings() []string {
	levels := make([]string, 0, len(stringToLevelMap))
	for k := range stringToLevelMap {
		levels = append(levels, k)
	}
	return levels
}
