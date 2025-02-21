package logger

import (
	"log"
	"os"
	"strings"
)

type defaultLogger struct {
	level string
}

func NewDefaultLogger() *defaultLogger {
	return &defaultLogger{
		level: LevelInfo,
	}
}

func (d *defaultLogger) SetLevel(level string) {
	d.level = strings.ToLower(level)
}

func (d *defaultLogger) shouldLog(messageLevel string) bool {
	levels := map[string]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
	}

	currentLevel := levels[d.level]
	msgLevel := levels[messageLevel]

	return msgLevel >= currentLevel
}

func (d *defaultLogger) Debug(msg string) {
	if d.shouldLog(LevelDebug) {
		log.Printf("[DEBUG] %s", msg)
	}
}

func (d *defaultLogger) Info(msg string) {
	if d.shouldLog(LevelInfo) {
		log.Printf("[INFO] %s", msg)
	}
}

func (d *defaultLogger) Warn(msg string) {
	if d.shouldLog(LevelWarn) {
		log.Printf("[WARN] %s", msg)
	}
}

func (d *defaultLogger) Error(msg string) {
	if d.shouldLog(LevelError) {
		log.Printf("[ERROR] %s", msg)
	}
}

func (d *defaultLogger) Fatal(msg string) {
	if d.shouldLog(LevelFatal) {
		log.Printf("[FATAL] %s", msg)
		os.Exit(1)
	}
}
