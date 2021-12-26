package webapi

import (
	"reflect"
	"testing"
)

func Test_ApiState_MustHaveMethod(t *testing.T) {
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
