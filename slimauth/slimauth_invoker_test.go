package slimauth

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/stretchr/testify/require"
)

func TestNewSlimAuthInvoker(t *testing.T) {
	h := NewSlimAuthApiHandler(SlimAuthApiHandlerOption{
		SecretFinder: finderForTest,
	})

	type plusReq struct{ X, Y int }
	h.RegisterMethod(webapi.ApiMethod{
		Name: "plus",
		Value: reflect.ValueOf(func(req plusReq) int {
			return req.X + req.Y
		}),
	})

	e := webapi.NewEngine()
	e.Handle("/{~method}", h, nil)
	s := httptest.NewServer(e)

	t.Run("ok", func(t *testing.T) {
		invoker := NewSlimAuthInvoker[plusReq, int](SlimAuthInvokerOp{
			Uri:    s.URL + "/plus",
			Key:    _key,
			Secret: _secret,
		})
		result, err := invoker.Do(plusReq{1, 2})
		require.NoError(t, err)
		require.Equal(t, 3, result)
	})

	t.Run("bad-key", func(t *testing.T) {
		invoker := NewSlimAuthInvoker[plusReq, int](SlimAuthInvokerOp{
			Uri:    s.URL + "/plus",
			Key:    "bad",
			Secret: _secret,
		})
		_, err := invoker.Do(plusReq{1, 2})
		require.Error(t, err)
		require.Regexp(t, `unknown key`, err.Error())
	})
}
