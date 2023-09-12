package webapi

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cmstar/go-errx"
)

func Test_basicApiMethodCaller_Call_WithoutMethod(t *testing.T) {
	defer func() {
		e := recover()
		if e == nil {
			t.Error("should panic")
			return
		}
	}()

	caller := NewBasicApiMethodCaller()
	caller.Call(&ApiState{})
}

func Test_basicApiMethodCaller_Call(t *testing.T) {
	caller := NewBasicApiMethodCaller()

	tests := []struct {
		name      string
		method    any
		args      []any
		data      any
		err       error
		deferFunc func(t *testing.T)
	}{
		{
			name:   "empty",
			method: func() {},
			args:   []any{},
			data:   nil,
			err:    nil,
		},

		{
			name:   "succ1",
			method: func(a, b int) struct{ a, b int } { return struct{ a, b int }{a, b} },
			args:   []any{1, 2},
			data:   struct{ a, b int }{1, 2},
			err:    nil,
		},

		{
			name:   "bizerr1",
			method: func(v []int) error { return errx.NewBizError(v[0]+v[1], "msg", nil) },
			args:   []any{[]int{1, 2}},
			data:   nil,
			err:    errx.NewBizError(3, "msg", nil),
		},

		{
			name:   "succ2",
			method: func(a, b int) (string, error) { return "succ", nil },
			args:   []any{1, 2},
			data:   "succ",
			err:    nil,
		},

		{
			name:   "bizerr2",
			method: func() (string, error) { return "data", errx.NewBizError(55, "m", nil) },
			args:   []any{},
			data:   "data",
			err:    errx.NewBizError(55, "m", nil),
		},

		{
			name:   "err1",
			method: func() error { return errors.New("err") },
			args:   []any{},
			data:   nil,
			err:    errors.New("err"),
		},

		{
			name:   "err2",
			method: func() (int, error) { return 3, errors.New("err") },
			args:   []any{},
			data:   3,
			err:    errors.New("err"),
		},

		{
			name:   "too-many-output",
			method: func() (int, int, error) { return 0, 0, nil },
			args:   []any{},
			data:   nil,
			err:    nil,
			deferFunc: func(t *testing.T) {
				checkRecoveredError(t, recover(), "the return value of method '' cannot be greater than 2")
			},
		},

		{
			name:   "wrong-output",
			method: func() (int, int) { return 0, 0 },
			args:   []any{},
			data:   nil,
			err:    nil,
			deferFunc: func(t *testing.T) {
				checkRecoveredError(t, recover(), "the second output parameter must be an error, got int")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.deferFunc != nil {
				defer tt.deferFunc(t)
			}

			argLen := len(tt.args)
			state := &ApiState{
				Method: ApiMethod{"", reflect.ValueOf(tt.method), ""},
				Args:   make([]reflect.Value, argLen),
			}

			for i := 0; i < argLen; i++ {
				state.Args[i] = reflect.ValueOf(tt.args[i])
			}

			caller.Call(state)

			if !reflect.DeepEqual(tt.data, state.Data) {
				t.Errorf("expected response %v, got %v", tt.data, state.Response)
			}

			compareError(t, tt.err, state.Error)
		})
	}
}

// 用于检测方法 panic 的 error 。 recovered 必须是个 error ，且错误信息为 expectedMessage 。
func checkRecoveredError(t *testing.T, recovered any, expectedMessage string) {
	if recovered == nil {
		t.Error("should panic")
		return
	}

	var err error
	var ok bool
	if err, ok = recovered.(error); !ok {
		t.Errorf("should panic an error, got %T", err)
		return
	}

	if err.Error() != expectedMessage {
		t.Errorf("expect error message '%s', got '%s'", expectedMessage, err.Error())
		return
	}
}

// checkPrefixError 用于 compareError 方法检测 got.Error() 的值，用于断言该值的开头部分。
// 应对错误信息很长或者不稳定的情况。
type checkPrefixError struct {
	prefix string
}

func (e checkPrefixError) Error() string { return e.prefix }

func compareError(t *testing.T, expected, got error) {
	if expected == nil {
		if got != nil {
			t.Errorf("expect no error, got %s", got)
		}

		return
	}

	if got == nil {
		t.Errorf("expect error '%s', got nil", expected)
		return
	}

	// Check checkPrefixError.
	if pErr, ok := expected.(checkPrefixError); ok {
		if !strings.HasPrefix(got.Error(), pErr.prefix) {
			t.Errorf("expect error starts with '%s', got %s", pErr.prefix, got)
		}
		return
	}

	// Check BizError.
	toBizErr := func(e error) errx.BizError {
		if be, ok := e.(errx.BizError); ok {
			return be
		}
		return nil
	}

	expectedBizErr, gotBizErr := toBizErr(expected), toBizErr(got)
	if expectedBizErr == nil {
		return
	}

	if gotBizErr == nil {
		t.Errorf("expect BizError, got %T", got)
		return
	}

	if expectedBizErr == nil {
		if gotBizErr == nil {
			t.Errorf("expect BizError, got %T", got)
			return
		}
	}

	if expectedBizErr.Code() != gotBizErr.Code() {
		t.Errorf("expect error-code %v, got %v", expectedBizErr.Code(), gotBizErr.Code())
	}

	if expectedBizErr.Message() != gotBizErr.Message() {
		t.Errorf("expect error-message '%v', got '%v'", expectedBizErr.Message(), gotBizErr.Message())
	}

	if expectedBizErr.Error() != gotBizErr.Error() {
		t.Errorf("expect error '%v', got '%v'", expected.Error(), gotBizErr.Error())
	}

	compareInnerError(t, expectedBizErr.Cause(), gotBizErr.Cause())
}

func compareInnerError(t *testing.T, expected, got error) {
	if expected == nil {
		if got != nil {
			t.Errorf("expect no inner-error, got '%s'", got)
		}

		return
	}

	if got == nil {
		t.Errorf("expect inner-error '%s', got nil", expected)
		return
	}

	if expected.Error() != got.Error() {
		t.Errorf("expect error '%s', got '%s'", expected, got)
	}
}
