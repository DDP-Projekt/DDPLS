package handlers

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/tliron/glsp"
)

func RecoverErr[T any](handler func(*glsp.Context, T) error) func(*glsp.Context, T) error {
	return func(context *glsp.Context, params T) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				stack_trace := fmt.Errorf("stack trace: %s", string(debug.Stack()))

				var recovered_err error
				switch recovered := recovered.(type) {
				case error:
					recovered_err = recovered
				default:
					recovered_err = fmt.Errorf("unknown panic value of type %s: %v", reflect.TypeOf(recovered), recovered)
				}

				if err != nil {
					err = errors.Join(err, recovered_err, stack_trace)
					return
				}
				err = errors.Join(recovered_err, stack_trace)
				return
			}
		}()
		return handler(context, params)
	}
}

func RecoverAnyErr[T, R any](handler func(*glsp.Context, T) (R, error)) func(*glsp.Context, T) (R, error) {
	return func(context *glsp.Context, params T) (result R, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				stack_trace := fmt.Errorf("stack trace: %s", string(debug.Stack()))

				var recovered_err error
				switch recovered := recovered.(type) {
				case error:
					recovered_err = recovered
				default:
					recovered_err = fmt.Errorf("unknown panic value of type %s: %v", reflect.TypeOf(recovered), recovered)
				}

				if err != nil {
					err = errors.Join(err, recovered_err, stack_trace)
					return
				}
				err = errors.Join(recovered_err, stack_trace)
				return
			}
		}()
		return handler(context, params)
	}
}
