package trading

type Logger interface {
	Debugf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Warningf(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	WithField(key string, value interface{}) Logger

	WithFields(fields map[string]interface{}) Logger
}
