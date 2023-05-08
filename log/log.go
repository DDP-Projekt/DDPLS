package log

import (
	"fmt"
	"runtime"

	"github.com/tliron/kutil/logging"
)

func fmtWithCaller(format string) string {
	pc, _, no, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		return fmt.Sprintf("[%s: %d] %s", details.Name(), no, format)
	}
	return fmt.Sprintf("[no info] %s", format)
}

var logger = logging.GetLogger("ddp.ddpls")

func Errorf(format string, args ...any) {
	logger.Errorf(fmtWithCaller(format), args...)
}

func Warningf(format string, args ...any) {
	logger.Warningf(fmtWithCaller(format), args...)
}

func Infof(format string, args ...any) {
	logger.Infof(fmtWithCaller(format), args...)
}
