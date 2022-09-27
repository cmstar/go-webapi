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
			"get", "post", "put", "delete", "patch", "head", "trace", "connect", "options",
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

	run("GET")
	run("POST")
	run("PUT")
	run("DELETE")
	run("PATCH")
	run("TRACE")
	run("HEAD")
	run("CONNECT")
	run("OPTIONS")
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

	run("GET", e.HandleGet)
	run("POST", e.HandlePost)
	run("PUT", e.HandlePut)
	run("DELETE", e.HandleDelete)
	run("PATCH", e.HandlePatch)
	run("TRACE", e.HandleTrace)
	run("HEAD", e.HandleHead)
	run("CONNECT", e.HandleConnect)
	run("OPTIONS", e.HandleOptions)
}
