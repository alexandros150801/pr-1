package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Level — уровень логирования.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "?"
	}
}

// Logger — логгер с уровнями и ротацией файла.
type Logger struct {
	minLevel Level
	out      *log.Logger
	file     io.Closer
}

var defaultLogger *Logger = &Logger{minLevel: LevelInfo, out: log.New(os.Stderr, "", 0)}

// InitLogger инициализирует глобальный логгер.
func InitLogger(cfg *Config) (*Logger, error) {
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	var closer io.Closer
	if cfg.Log.File != "" {
		lj := &lumberjack.Logger{
			Filename:   cfg.Log.File,
			MaxSize:    cfg.Log.MaxSizeMB,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     30,
			LocalTime:  true,
		}
		writers = append(writers, lj)
		closer = lj
	}

	lvl := LevelInfo
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		lvl = LevelDebug
	case "info":
		lvl = LevelInfo
	case "warn", "warning":
		lvl = LevelWarn
	case "error":
		lvl = LevelError
	}

	l := &Logger{
		minLevel: lvl,
		out:      log.New(io.MultiWriter(writers...), "", 0),
		file:     closer,
	}
	defaultLogger = l
	l.Info("логгер инициализирован")
	return l, nil
}

// Default возвращает глобальный логгер.
func Default() *Logger { return defaultLogger }

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.out.Printf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05.000"), level, msg)
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(LevelError, format, args...) }

// Close закрывает файл лога.
func (l *Logger) Close() {
	if l.file != nil {
		_ = l.file.Close()
	}
}
