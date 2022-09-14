package webapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiState_MustHaveMethod(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		defer func() {
			e := recover()
			if e == nil {
				t.Error("should panic")
				return
			}
		}()

		(&ApiState{}).MustHaveMethod()
	})

	t.Run("WrongType", func(t *testing.T) {
		defer func() {
			e := recover()
			if e == nil {
				t.Error("should panic")
				return
			}
		}()

		(&ApiState{Method: ApiMethod{"", reflect.ValueOf(1), ""}}).MustHaveMethod()
	})

	t.Run("OK", func(t *testing.T) {
		defer func() {
			e := recover()
			if e != nil {
				t.Error("should not panic")
				return
			}
		}()

		f := reflect.ValueOf(func() {})
		(&ApiState{Method: ApiMethod{"", f, ""}}).MustHaveMethod()
	})
}

func TestApiState_SetCustomData(t *testing.T) {
	type k1 int
	type k2 int

	state := &ApiState{}
	state.SetCustomData(k1(0), 1)
	state.SetCustomData(k2(0), 2)

	_, ok := state.GetCustomData(0)
	assert.False(t, ok)

	v, ok := state.GetCustomData(k1(0))
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	v, ok = state.GetCustomData(k2(0))
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}
