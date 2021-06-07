package logrus

import (
	"github.com/lukasz-zimnoch/dexly/trading"
	"github.com/sirupsen/logrus"
	"os"
)

type wrapper struct {
	*logrus.Entry
}

func (w *wrapper) WithField(key string, value interface{}) trading.Logger {
	return &wrapper{w.Entry.WithField(key, value)}
}

func (w *wrapper) WithFields(fields map[string]interface{}) trading.Logger {
	return &wrapper{w.Entry.WithFields(fields)}
}

func ConfigureStandardLogger(format, level string) trading.Logger {
	fieldMap := logrus.FieldMap{
		logrus.FieldKeyLevel: "severity",
		logrus.FieldKeyMsg:   "message",
	}

	if format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			FieldMap: fieldMap,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			FieldMap:      fieldMap,
		})
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Fatalf("could not parse log level: [%v]", err)
	}

	logrus.SetLevel(logLevel)

	logrus.SetOutput(os.Stdout)

	return &wrapper{
		logrus.StandardLogger().WithFields(map[string]interface{}{}),
	}
}
