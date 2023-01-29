package slimapi

import (
	"net/http/httptest"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-webapi"
	"github.com/stretchr/testify/require"
)

func TestSlimApiInvoker_Do(t *testing.T) {
	e := webapi.NewEngine()
	e.Handle("/{~method}", handlerForIntegrationTest, nil)
	s := httptest.NewServer(e)

	t.Run("ok", func(t *testing.T) {
		invoker := NewSlimApiInvoker[PlusRequest, int](s.URL + "/Plus")
		b := 2
		result, err := invoker.Do(PlusRequest{
			A: 101,
			B: &b,
		})
		require.NoError(t, err)
		require.Equal(t, 103, result)
	})

	t.Run("biz", func(t *testing.T) {
		invoker := NewSlimApiInvoker[ShowErrorRequest, string](s.URL + "/ShowError")
		result, err := invoker.Do(ShowErrorRequest{
			Type: ShowError_BizError999,
		})
		require.Error(t, err)

		bizErr, ok := err.(errx.BizError)
		require.True(t, ok)
		require.Equal(t, 999, bizErr.Code())
		require.Equal(t, "", result)
		require.NotNil(t, bizErr.Cause())
		require.Regexp(t, `request ".+?/ShowError": \(999\)`, bizErr.Cause().Error())
	})

	t.Run("bad", func(t *testing.T) {
		invoker := NewSlimApiInvoker[int, int]("bad-url")
		_, err := invoker.Do(1)
		require.Error(t, err)
		require.Regexp(t, `request "bad-url":`, err.Error())
	})
}

func TestSlimApiInvoker_MustDo(t *testing.T) {
	e := webapi.NewEngine()
	e.Handle("/{~method}", handlerForIntegrationTest, nil)
	s := httptest.NewServer(e)

	t.Run("ok", func(t *testing.T) {
		invoker := NewSlimApiInvoker[PlusRequest, int](s.URL + "/Plus")
		b := 2
		result := invoker.MustDo(PlusRequest{
			A: 101,
			B: &b,
		})
		require.Equal(t, 103, result)
	})

	t.Run("panic", func(t *testing.T) {
		defer func() {
			err := recover()
			require.NotNil(t, err)

			bizErr, ok := err.(errx.BizError)
			require.True(t, ok)
			require.Equal(t, 999, bizErr.Code())
		}()

		invoker := NewSlimApiInvoker[ShowErrorRequest, string](s.URL + "/ShowError")
		invoker.MustDo(ShowErrorRequest{
			Type: ShowError_BizError999,
		})

		require.Fail(t, "should not run")
	})
}
