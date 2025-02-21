package logger

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Debug(msg string)
	Fatal(msg string)
	SetLevel(level string)
}

var currentLogger Logger = NewDefaultLogger()

func SetLogger(custom Logger) {
	currentLogger = custom
}

func SetLevel(level string) {
	currentLogger.SetLevel(level)
}

func Info(msg string) {
	currentLogger.Info(msg)
}

func Warn(msg string) {
	currentLogger.Warn(msg)
}

func Error(msg string) {
	currentLogger.Error(msg)
}

func Debug(msg string) {
	currentLogger.Debug(msg)
}

func Fatal(msg string) {
	currentLogger.Fatal(msg)
}
