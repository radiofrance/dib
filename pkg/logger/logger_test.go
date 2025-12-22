package logger_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Parallel()

	logger.Infof("this is info")
	logger.Debugf("should not be displayed")

	debugLvl := "debug"
	logger.SetLevel(&debugLvl)
	logger.Debugf("should be displayed")
	assert.Equal(t, logger.LogLevelDebug, logger.Get().Level)

	logger.Warnf("this is a warning")
	logger.Errorf("this is an error")
}
