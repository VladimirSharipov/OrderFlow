package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger структура для логирования
type Logger struct {
	*logrus.Logger
}

// Config конфигурация логгера
type Config struct {
	Level  string `env:"LOG_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"json"`
}

// New создает новый логгер
func New(config Config) *Logger {
	logger := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(strings.ToLower(config.Level))
	if err != nil {
		logger.Warnf("Invalid log level %s, using info", config.Level)
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Устанавливаем формат
	switch strings.ToLower(config.Format) {
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Устанавливаем вывод в stdout
	logger.SetOutput(os.Stdout)

	return &Logger{Logger: logger}
}

// WithField создает новую запись с полем
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields создает новую запись с полями
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError создает новую запись с ошибкой
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Default создает логгер с настройками по умолчанию
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "json",
	})
}
