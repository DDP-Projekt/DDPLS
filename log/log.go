package log

import "github.com/tliron/kutil/logging"

var logger = logging.GetLogger("ddp.ddpls")

func Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

func Warningf(format string, args ...any) {
	logger.Warningf(format, args...)
}

func Infof(format string, args ...any) {
	logger.Infof(format, args...)
}
