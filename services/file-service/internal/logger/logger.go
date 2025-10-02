package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	Log = NewLogger("info")
}

// NewLogger creates a new structured logger instance
func NewLogger(level string) *logrus.Logger {
	logger := logrus.New()

	// Set log format to JSON for production, text for development
	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Output to stdout
	logger.SetOutput(os.Stdout)

	return logger
}

// WithRequestID adds request ID to logger fields
func WithRequestID(logger *logrus.Logger, requestID string) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

// WithUserID adds user ID to logger fields
func WithUserID(logger *logrus.Logger, userID string) *logrus.Entry {
	return logger.WithField("user_id", userID)
}

// WithFileID adds file ID to logger fields
func WithFileID(logger *logrus.Logger, fileID string) *logrus.Entry {
	return logger.WithField("file_id", fileID)
}
