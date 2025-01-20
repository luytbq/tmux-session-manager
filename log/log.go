package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luytbq/tmux-session-manager/common"
	"github.com/luytbq/tmux-session-manager/config"
)

var (
	LevelTrace = common.LogLevelTrace
	LevelDebug = common.LogLevelDebug
	LevelInfo  = common.LogLevelInfo
	LevelWarn  = common.LogLevelWarn
	LevelError = common.LogLevelError
	LevelFatal = common.LogLevelFatal
)

const baseLevel = config.LogLevel

var logFile string

func init() {
	file, err := getLogFile()
	if err != nil {
		panic(err)
	}
	logFile = file
}

func Trace(objs ...any) {
	Log(LevelTrace, objs...)
}
func Debug(objs ...any) {
	Log(LevelDebug, objs...)
}
func Info(objs ...any) {
	Log(LevelInfo, objs...)
}
func Warn(objs ...any) {
	Log(LevelWarn, objs...)
}
func Error(objs ...any) {
	Log(LevelError, objs...)
}
func Fatal(objs ...any) {
	Log(LevelFatal, objs...)
}

func Log(level int, objs ...any) {
	if level < baseLevel {
		return
	}

	logTime := formatTime(time.Now())

	var msg = fmt.Sprintf("%s %s ", logTime, getLogLevelLabel(level))

	for i, obj := range objs {
		if i > 1 {
			msg += fmt.Sprintf("\n%+v ", obj)
		} else {
			msg += fmt.Sprintf("%+v ", obj)
		}
	}

	// Ensure the message ends with a newline
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg += "\n"
	}

	err := appendFile(logFile, msg)
	if err != nil {
		stdOut(err.Error())
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func getLogLevelLabel(level int) string {
	switch int(level) {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOW LEVEL"
	}
}

func getLogFile() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(configDir, config.AppName)
	if err := os.MkdirAll(appDir, os.ModePerm); err != nil {
		return "", err
	}

	file := filepath.Join(appDir, config.LogFile)
	_, err = os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(file)
	}

	return file, err
}

// Append file content
func appendFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func stdOut(msg string) {
	fmt.Fprintf(os.Stdout, "%s\r\n", msg)
}
