/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

// Package log provides a minimal logging framework.
package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ontio/ontology-rosetta/process"
)

const (
	Blue   = "0;34"
	Red    = "0;31"
	Green  = "0;32"
	Yellow = "0;33"
	Cyan   = "0;36"
	Pink   = "1;35"
)

func Color(code, msg string) string {
	return fmt.Sprintf("\033[%sm%s\033[m", code, msg)
}

const (
	TraceLog = iota
	DebugLog
	InfoLog
	WarnLog
	ErrorLog
	FatalLog
	MaxLevelLog
)

var (
	levels = map[int]string{
		DebugLog: Color(Green, "[DEBUG]"),
		InfoLog:  Color(Cyan, "[INFO ]"),
		WarnLog:  Color(Yellow, "[WARN ]"),
		ErrorLog: Color(Red, "[ERROR]"),
		FatalLog: Color(Red, "[FATAL]"),
		TraceLog: Color(Pink, "[TRACE]"),
	}
	Stdout = os.Stdout
)

const (
	namePrefix = "LEVEL"
	callDepth  = 2
	maxLogSize = 20
	megabyte   = 1024 * 1024
)

func GetGID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

var RosettaLog *Logger

func init() {
	//Default print to console
	InitLog(InfoLog, Stdout)
}

func LevelName(level int) string {
	if name, ok := levels[level]; ok {
		return name
	}
	return namePrefix + strconv.Itoa(level)
}

func NameLevel(name string) int {
	for k, v := range levels {
		if v == name {
			return k
		}
	}
	var level int
	if strings.HasPrefix(name, namePrefix) {
		level, _ = strconv.Atoi(name[len(namePrefix):])
	}
	return level
}

type Logger struct {
	level   int
	logger  *log.Logger
	logFile *os.File
}

func New(out io.Writer, prefix string, flag, level int, file *os.File) *Logger {
	return &Logger{
		level:   level,
		logger:  log.New(out, prefix, flag),
		logFile: file,
	}
}

func (l *Logger) SetDebugLevel(level int) error {
	if level > MaxLevelLog || level < 0 {
		return errors.New("invalid debug level")
	}

	l.level = level
	return nil
}

func (l *Logger) Output(level int, a ...interface{}) error {
	if level >= l.level {
		gid := GetGID()
		gidStr := strconv.FormatUint(gid, 10)

		a = append([]interface{}{LevelName(level), "GID",
			gidStr + ","}, a...)

		return l.logger.Output(callDepth, fmt.Sprintln(a...))
	}
	return nil
}

func (l *Logger) Outputf(level int, format string, v ...interface{}) error {
	if level >= l.level {
		gid := GetGID()
		v = append([]interface{}{LevelName(level), "GID",
			gid}, v...)

		return l.logger.Output(callDepth, fmt.Sprintf("%s %s %d, "+format+"\n", v...))
	}
	return nil
}

func (l *Logger) Trace(a ...interface{}) {
	l.Output(TraceLog, a...)
}

func (l *Logger) Tracef(format string, a ...interface{}) {
	l.Outputf(TraceLog, format, a...)
}

func (l *Logger) Debug(a ...interface{}) {
	l.Output(DebugLog, a...)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Outputf(DebugLog, format, a...)
}

func (l *Logger) Info(a ...interface{}) {
	l.Output(InfoLog, a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Outputf(InfoLog, format, a...)
}

func (l *Logger) Warn(a ...interface{}) {
	l.Output(WarnLog, a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Outputf(WarnLog, format, a...)
}

func (l *Logger) Warningf(format string, a ...interface{}) {
	l.Outputf(WarnLog, format, a...)
}

func (l *Logger) Error(a ...interface{}) {
	l.Output(ErrorLog, a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Outputf(ErrorLog, format, a...)
}

func (l *Logger) Fatal(a ...interface{}) {
	l.Output(FatalLog, a...)
	process.Exit(1)
}

func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.Outputf(FatalLog, format, a...)
	process.Exit(1)
}

func Trace(a ...interface{}) {
	if TraceLog < RosettaLog.level {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")

	a = append([]interface{}{funcName + "()", fileName + ":" + strconv.Itoa(line)}, a...)

	RosettaLog.Trace(a...)
}

func Tracef(format string, a ...interface{}) {
	if TraceLog < RosettaLog.level {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")

	a = append([]interface{}{funcName, fileName, line}, a...)

	RosettaLog.Tracef("%s() %s:%d "+format, a...)
}

func Debug(a ...interface{}) {
	if DebugLog < RosettaLog.level {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	a = append([]interface{}{f.Name(), fileName + ":" + strconv.Itoa(line)}, a...)

	RosettaLog.Debug(a...)
}

func Debugf(format string, a ...interface{}) {
	if DebugLog < RosettaLog.level {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	a = append([]interface{}{f.Name(), fileName, line}, a...)

	RosettaLog.Debugf("%s %s:%d "+format, a...)
}

func Info(a ...interface{}) {
	RosettaLog.Info(a...)
}

func Warn(a ...interface{}) {
	RosettaLog.Warn(a...)
}

func Error(a ...interface{}) {
	RosettaLog.Error(a...)
}

func Fatal(a ...interface{}) {
	RosettaLog.Fatal(a...)
}

func Infof(format string, a ...interface{}) {
	RosettaLog.Infof(format, a...)
}

func Warnf(format string, a ...interface{}) {
	RosettaLog.Warnf(format, a...)
}

func Errorf(format string, a ...interface{}) {
	RosettaLog.Errorf(format, a...)
}

func Fatalf(format string, a ...interface{}) {
	RosettaLog.Fatalf(format, a...)
}

// used for develop stage and not allowed in production enforced by CI

var Test = Fatal
var Testf = Fatalf

func FileOpen(path string) (*os.File, error) {
	if fi, err := os.Stat(path); err == nil {
		if !fi.IsDir() {
			return nil, fmt.Errorf("open %s: not a directory", path)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0766); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	var currenttime = time.Now().Format("2006-01-02_15.04.05")

	logfile, err := os.OpenFile(path+currenttime+"_LOG.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return logfile, nil
}

//Init deprecated, use InitLog instead
func Init(a ...interface{}) {
	os.Stderr.WriteString("warning: use of deprecated Init. Use InitLog instead\n")
	InitLog(InfoLog, a...)
}

func InitLog(logLevel int, a ...interface{}) {
	writers := []io.Writer{}
	var logFile *os.File
	var err error
	if len(a) == 0 {
		writers = append(writers, ioutil.Discard)
	} else {
		for _, o := range a {
			switch o := o.(type) {
			case string:
				logFile, err = FileOpen(o)
				if err != nil {
					fmt.Println("error: open log file failed")
					os.Exit(1)
				}
				writers = append(writers, logFile)
			case *os.File:
				writers = append(writers, o)
			default:
				fmt.Println("error: invalid log location")
				os.Exit(1)
			}
		}
	}
	fileAndStdoutWrite := io.MultiWriter(writers...)
	RosettaLog = New(fileAndStdoutWrite, "", log.LUTC|log.Ldate|log.Lmicroseconds, logLevel, logFile)
}

func GetLogFileSize() (int64, error) {
	f, e := RosettaLog.logFile.Stat()
	if e != nil {
		return 0, e
	}
	return f.Size(), nil
}

func GetMaxLogChangeInterval(maxLogSize int64) int64 {
	if maxLogSize != 0 {
		return maxLogSize * megabyte
	} else {
		return maxLogSize * megabyte
	}
}

func CheckIfNeedNewFile() bool {
	logFileSize, err := GetLogFileSize()
	maxLogFileSize := GetMaxLogChangeInterval(0)
	if err != nil {
		return false
	}
	if logFileSize > maxLogFileSize {
		return true
	} else {
		return false
	}
}

func ClosePrintLog() error {
	var err error
	if RosettaLog.logFile != nil {
		err = RosettaLog.logFile.Close()
	}
	return err
}
