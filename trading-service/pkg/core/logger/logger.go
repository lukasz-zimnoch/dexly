package logger

import "github.com/sirupsen/logrus"

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func WithFields(fields map[string]interface{}) Logger {
	return logrus.WithFields(fields)
}
