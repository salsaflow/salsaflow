package log

import (
	// Stdlib
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"

	// Vendor
	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
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

var lock sync.Mutex

var logWriter io.Writer

func init() {
	Replace(os.Stderr)
}

func Replace(newWriter io.Writer) (formerWriter io.Writer) {
	lock.Lock()
	formerWriter = logWriter
	logWriter = ansicolor.NewAnsiColorWriter(newWriter)
	lock.Unlock()
	return
}

func Disable() {
	lock.Lock()
	logWriter = ioutil.Discard
	lock.Unlock()
}

var v Level = Info

func SetV(level Level) {
	atomic.StoreUint32((*uint32)(&v), uint32(level))
}

func V(level Level) Logger {
	if threshold := atomic.LoadUint32((*uint32)(&v)); threshold > uint32(level) {
		return Logger(false)
	}
	return Logger(true)
}

func (l Logger) log(v ...interface{}) {
	if l {
		lock.Lock()
		defer lock.Unlock()
		fmt.Fprint(logWriter, v...)
	}
}

func (l Logger) unsafeLog(v ...interface{}) {
	if l {
		fmt.Fprint(logWriter, v...)
	}
}

func (l Logger) logf(format string, v ...interface{}) {
	if l {
		lock.Lock()
		defer lock.Unlock()
		fmt.Fprintf(logWriter, format, v...)
	}
}

func (l Logger) unsafeLogf(format string, v ...interface{}) {
	if l {
		fmt.Fprintf(logWriter, format, v...)
	}
}

func (l Logger) logln(v ...interface{}) {
	if l {
		lock.Lock()
		defer lock.Unlock()
		fmt.Fprintln(logWriter, v...)
	}
}

func (l Logger) unsafeLogln(v ...interface{}) {
	if l {
		fmt.Fprintln(logWriter, v...)
	}
}

func (l Logger) Lock() {
	lock.Lock()
}

func (l Logger) Unlock() {
	lock.Unlock()
}

func (l Logger) Print(v ...interface{}) {
	l.log(v...)
}

func (l Logger) UnsafePrint(v ...interface{}) {
	l.unsafeLog(v...)
}

func (l Logger) Printf(format string, v ...interface{}) {
	l.logf(format, v...)
}

func (l Logger) UnsafePrintf(format string, v ...interface{}) {
	l.unsafeLogf(format, v...)
}

func (l Logger) Println(v ...interface{}) {
	l.logln(v...)
}

func (l Logger) UnsafePrintln(v ...interface{}) {
	l.unsafeLogln(v...)
}

func (l Logger) Fatal(v ...interface{}) {
	l.log(v...)
	os.Exit(1)
}

func (l Logger) UnsafeFatal(v ...interface{}) {
	l.unsafeLog(v...)
	os.Exit(1)
}

func (l Logger) Fatalf(format string, v ...interface{}) {
	l.logf(format, v...)
	os.Exit(1)
}

func (l Logger) UnsafeFatalf(format string, v ...interface{}) {
	l.unsafeLogf(format, v...)
	os.Exit(1)
}

func (l Logger) Fatalln(v ...interface{}) {
	l.logln(v...)
	os.Exit(1)
}

func (l Logger) UnsafeFatalln(v ...interface{}) {
	l.unsafeLogln(v...)
	os.Exit(1)
}

func (l Logger) Run(msg string) {
	l.logf("[RUN]      %v\n", msg)
}

func (l Logger) UnsafeRun(msg string) {
	l.unsafeLogf("[RUN]      %v\n", msg)
}

func (l Logger) Skip(msg string) {
	l.logf("[SKIP]     %v\n", msg)
}

func (l Logger) UnsafeSkip(msg string) {
	l.unsafeLogf("[SKIP]     %v\n", msg)
}

func (l Logger) Warn(msg string) {
	l.logf("%v     %v\n", color.YellowString("[WARN]"), msg)
}

func (l Logger) UnsafeWarn(msg string) {
	l.unsafeLogf("%v     %v\n", color.YellowString("[WARN]"), msg)
}

func (l Logger) Go(msg string) {
	l.logf("[GO]       %v\n", msg)
}

func (l Logger) UnsafeGo(msg string) {
	l.unsafeLogf("[GO]       %v\n", msg)
}

func (l Logger) Log(msg string) {
	l.logf("[LOG]      %v\n", msg)
}

func (l Logger) UnsafeLog(msg string) {
	l.unsafeLogf("[LOG]      %v\n", msg)
}

func (l Logger) Ok(msg string) {
	l.logf("[OK]       %v\n", msg)
}

func (l Logger) UnsafeOk(msg string) {
	l.unsafeLogf("[OK]       %v\n", msg)
}

func (l Logger) Fail(msg string) {
	l.logf("%v     %v\n", color.RedString("[FAIL]"), msg)
}

func (l Logger) UnsafeFail(msg string) {
	l.unsafeLogf("%v     %v\n", color.RedString("[FAIL]"), msg)
}

func (l Logger) Rollback(msg string) {
	l.logf("[ROLLBACK] %v\n", msg)
}

func (l Logger) UnsafeRollback(msg string) {
	l.unsafeLogf("[ROLLBACK] %v\n", msg)
}

func (l Logger) NewLine(msg string) {
	l.logf("           %v\n", msg)
}

func (l Logger) UnsafeNewLine(msg string) {
	l.unsafeLogf("           %v\n", msg)
}

func (l Logger) Stderr(stderr string) {
	if stderr != "" {
		lock.Lock()
		defer lock.Unlock()
		l.UnsafePrintln("<<<<< stderr")
		l.UnsafePrint(stderr)
		l.UnsafePrintln(">>>>> stderr")
	}
}

func (l Logger) UnsafeStderr(stderr string) {
	if stderr != "" {
		l.UnsafePrintln("<<<<< stderr")
		l.UnsafePrint(stderr)
		l.UnsafePrintln(">>>>> stderr")
	}
}

func (l Logger) Die(msg string, err error) {
	lock.Lock()
	defer lock.Unlock()
	l.UnsafeFail(msg)
	l.UnsafeFatalln("\nError:", err)
}

func (l Logger) UnsafeDie(msg string, err error) {
	l.UnsafeFail(msg)
	l.UnsafeFatalln("\nError:", err)
}

func (l Logger) FailWithDetails(msg string, details string) {
	if msg != "" {
		l.Fail(msg)
	}
	l.Stderr(details)
}

func Print(v ...interface{}) {
	V(Info).Print(v...)
}

func UnsafePrint(v ...interface{}) {
	V(Info).UnsafePrint(v...)
}

func Printf(format string, v ...interface{}) {
	V(Info).Printf(format, v...)
}

func UnsafePrintf(format string, v ...interface{}) {
	V(Info).UnsafePrintf(format, v...)
}

func Println(v ...interface{}) {
	V(Info).Println(v...)
}

func UnsafePrintln(v ...interface{}) {
	V(Info).UnsafePrintln(v...)
}

func Fatal(v ...interface{}) {
	V(Info).Fatal(v...)
}

func UnsafeFatal(v ...interface{}) {
	V(Info).UnsafeFatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	V(Info).Fatalf(format, v...)
}

func UnsafeFatalf(format string, v ...interface{}) {
	V(Info).UnsafeFatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	V(Info).Fatalln(v...)
}

func UnsafeFatalln(v ...interface{}) {
	V(Info).UnsafeFatalln(v...)
}

func Run(msg string) {
	V(Info).Run(msg)
}

func UnsafeRun(msg string) {
	V(Info).UnsafeRun(msg)
}

func Log(msg string) {
	V(Info).Log(msg)
}

func UnsafeLog(msg string) {
	V(Info).UnsafeLog(msg)
}

func Skip(msg string) {
	V(Info).Run(msg)
}

func UnsafeSkip(msg string) {
	V(Info).UnsafeSkip(msg)
}

func Go(msg string) {
	V(Info).Go(msg)
}

func UnsafeGo(msg string) {
	V(Info).UnsafeGo(msg)
}

func Ok(msg string) {
	V(Info).Ok(msg)
}

func UnsafeOk(msg string) {
	V(Info).UnsafeOk(msg)
}

func Warn(msg string) {
	V(Info).Warn(msg)
}

func UnsafeWarn(msg string) {
	V(Info).UnsafeWarn(msg)
}

func Fail(msg string) {
	V(Info).Fail(msg)
}

func UnsafeFail(msg string) {
	V(Info).UnsafeFail(msg)
}

func Rollback(msg string) {
	V(Info).Rollback(msg)
}

func UnsafeRollback(msg string) {
	V(Info).UnsafeRollback(msg)
}

func NewLine(msg string) {
	V(Info).NewLine(msg)
}

func UnsafeNewLine(msg string) {
	V(Info).UnsafeNewLine(msg)
}

func FailWithDetails(msg string, details string) {
	V(Info).FailWithDetails(msg, details)
}

func Stderr(stderr string) {
	V(Info).Stderr(stderr)
}

func UnsafeStderr(stderr string) {
	V(Info).UnsafeStderr(stderr)
}

func Die(msg string, err error) {
	V(Info).Die(msg, err)
}

func UnsafeDie(msg string, err error) {
	V(Info).UnsafeDie(msg, err)
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
