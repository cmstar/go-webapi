package webapi

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeFuncPipeline(t *testing.T) {
	decodeInt := func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
		if argType.Kind() != reflect.Int {
			return false, nil, nil
		}
		return true, 11, nil
	}
	decodeString := func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
		if argType.Kind() != reflect.String {
			return false, nil, nil
		}
		return true, "value", nil
	}
	errorOnFloat64 := func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
		if argType.Kind() != reflect.Float64 {
			return false, nil, nil
		}
		return false, nil, errors.New("e")
	}
	nilOnFloat32 := func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
		if argType.Kind() != reflect.Float32 {
			return false, nil, nil
		}
		return true, nil, nil
	}
	decoder := NewArgumentDecoderPipeline(
		ArgumentDecodeFunc(decodeInt),
		ArgumentDecodeFunc(decodeString),
		ArgumentDecodeFunc(errorOnFloat64),
		ArgumentDecodeFunc(nilOnFloat32),
	)

	run := func(fn any) *ApiState {
		s := &ApiState{
			Method: ApiMethod{
				Value: reflect.ValueOf(fn),
			},
		}
		decoder.Decode(s)
		return s
	}

	t.Run("empty", func(t *testing.T) {
		s := run(func() {})
		assert.Equal(t, 0, len(s.Args))
	})

	t.Run("state-ptr", func(t *testing.T) {
		s := run(func(*ApiState) {})
		assert.Equal(t, 1, len(s.Args))
		assert.Equal(t, s, s.Args[0].Interface())
	})

	t.Run("panic-state-value", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Equal(t, "method '' arg0 webapi.ApiState: must be a pointer", r.(error).Error())
		}()
		run(func(ApiState) {})
	})

	t.Run("string-int", func(t *testing.T) {
		s := run(func(string, int) {})
		assert.Equal(t, 2, len(s.Args))
		assert.Equal(t, "value", s.Args[0].Interface())
		assert.Equal(t, 11, s.Args[1].Interface())
	})

	t.Run("error", func(t *testing.T) {
		s := run(func(string, float64) {})
		assert.Equal(t, 0, len(s.Args))
		assert.NotNil(t, s.Error)
		assert.Equal(t, "e", s.Error.Error())
	})

	t.Run("panic-duplicate-type", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Equal(t, "method '' arg1 string: argument type cannot be duplicated", r.(error).Error())
		}()
		run(func(string, string) {})
	})

	t.Run("panic-unsupported-type", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Equal(t, "method '' arg0 int32: not supported", r.(error).Error())
		}()
		run(func(int32) {})
	})

	t.Run("panic-nil", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Equal(t, "method '' arg0 float32: value is nil", r.(error).Error())
		}()
		run(func(float32) {})
	})
}
