package helm

import "log"

type leveledLogger struct{}

func newLeveledLogger() *leveledLogger {
	return &leveledLogger{}
}

func (l *leveledLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Println(msg, keysAndValues)
}
func (l *leveledLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *leveledLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *leveledLogger) Warn(msg string, keysAndValues ...interface{})  {}
