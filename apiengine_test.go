package webapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApiEngine_Handle(t *testing.T) {
	h := setupApiHandlerWrapper(&ApiHandlerWrapper{
		HttpMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodTrace,
			http.MethodHead,
			http.MethodConnect,
			http.MethodOptions,
		},

		// 直接将 HTTP METHOD 赋值到 X-Method 头。解决 HEAD 方法不支持 body 的情况。
		ApiResponseWriter: ApiResponseWriterFunc(func(state *ApiState) {
			state.RawResponse.Header().Set("X-Method", state.RawRequest.Method)
		}),
	})
	e := NewEngine()
	e.Handle("/", h, nil)

	ts := httptest.NewServer(e)
	run := func(httpMethod string) {
		t.Run(httpMethod, func(t *testing.T) {
			req, _ := http.NewRequest(httpMethod, ts.URL, nil)
			res, _ := new(http.Client).Do(req)
			head, ok := res.Header["X-Method"]
			require.True(t, ok)
			require.Equal(t, httpMethod, head[0])
		})
	}

	run(http.MethodGet)
	run(http.MethodPost)
	run(http.MethodPut)
	run(http.MethodDelete)
	run(http.MethodPatch)
	run(http.MethodTrace)
	run(http.MethodHead)
	run(http.MethodConnect)
	run(http.MethodOptions)
}

func TestApiEngine_HandleActions(t *testing.T) {
	e := NewEngine()
	ts := httptest.NewServer(e)

	run := func(httpMethod string, handle func(path string, handlerFunc http.HandlerFunc)) {
		// 请求路径 = HTTP METHOD = X-Method 头
		t.Run(httpMethod, func(t *testing.T) {
			// path := "/test/" + httpMethod
			path := "/"

			handle(path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Method", r.Method)
			})

			req, _ := http.NewRequest(httpMethod, ts.URL, nil)
			res, _ := new(http.Client).Do(req)
			head, ok := res.Header["X-Method"]
			require.True(t, ok)
			require.Equal(t, httpMethod, head[0])
		})
	}

	run(http.MethodGet, e.HandleGet)
	run(http.MethodPost, e.HandlePost)
	run(http.MethodPut, e.HandlePut)
	run(http.MethodDelete, e.HandleDelete)
	run(http.MethodPatch, e.HandlePatch)
	run(http.MethodTrace, e.HandleTrace)
	run(http.MethodHead, e.HandleHead)
	run(http.MethodConnect, e.HandleConnect)
	run(http.MethodOptions, e.HandleOptions)
}
