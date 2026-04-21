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

	t.Run("error on streaming response", func(t *testing.T) {
		invoker := NewSlimApiInvoker[struct{}, string](s.URL + "/ServerSendEventWithError")
		_, err := invoker.Do(struct{}{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "streaming response text/event-stream, use DoRawStream/MustDoStream instead")
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

func TestSlimApiInvoker_DoRawStream(t *testing.T) {
	e := webapi.NewEngine()
	e.Handle("/{~method}", handlerForIntegrationTest, nil)
	s := httptest.NewServer(e)

	t.Run("sse", func(t *testing.T) {
		invoker := NewSlimApiInvoker[struct{}, string](s.URL + "/ServerSendEventWithError")
		seq := invoker.DoRawStream(struct{}{})
		var got []string
		for v, err := range seq {
			require.NoError(t, err)
			if v.Code == webapi.EventStreamEndCode {
				break
			}
			if v.Code != 0 {
				require.Equal(t, 500, v.Code)
				require.Equal(t, "error data", v.Data)
				break
			}
			got = append(got, v.Data)
		}
		require.Equal(t, []string{"a", "b"}, got)
	})

	t.Run("ndjson", func(t *testing.T) {
		invoker := NewSlimApiInvoker[struct{}, string](s.URL + "/NdJsonWithError")
		seq := invoker.DoRawStream(struct{}{})
		var got []string
		for v, err := range seq {
			require.NoError(t, err)
			if v.Code == webapi.EventStreamEndCode {
				break
			}
			if v.Code != 0 {
				require.Equal(t, 500, v.Code)
				require.Equal(t, "error data", v.Data)
				break
			}
			got = append(got, v.Data)
		}
		require.Equal(t, []string{"a", "b"}, got)
	})

	t.Run("non streaming endpoint", func(t *testing.T) {
		invoker := NewSlimApiInvoker[PlusRequest, int](s.URL + "/Plus")
		b := 2
		seq := invoker.DoRawStream(PlusRequest{A: 1, B: &b})
		n := 0
		for v, err := range seq {
			require.NoError(t, err)
			require.Equal(t, 0, v.Code)
			require.Equal(t, 3, v.Data)
			n++
		}
		require.Equal(t, 1, n)
	})
}
