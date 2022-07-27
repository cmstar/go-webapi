package slimapi

import (
	"io"
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_slimApiResponseWriter_WriteResponse(t *testing.T) {
	type args struct {
		response         *webapi.ApiResponse[any]
		callback         string
		wantBody         string
		wantPanicPattern string
	}

	instance := NewSlimApiApiResponseWriter()
	testOne := func(name string, a args) {
		t.Run(name, func(t *testing.T) {
			if a.wantPanicPattern != "" {
				defer func() {
					recovered := recover()
					require.NotNil(t, recovered)

					err, ok := recovered.(webapi.ApiError)
					require.True(t, ok, "must be a webapi.ApiError, got %T", recovered)

					assert.Regexp(t, a.wantPanicPattern, err.Error())
				}()
			}

			state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, "/", webapitest.NewStateSetup{})
			state.Response = a.response
			if a.callback != "" {
				setCallback(state, a.callback)
			}
			instance.WriteResponse(state)

			body, err := io.ReadAll(state.ResponseBody)
			require.NoError(t, err)
			assert.Equal(t, a.wantBody, string(body))
		})
	}

	testOne("empty", args{
		response: &webapi.ApiResponse[any]{
			Code:    0,
			Message: "",
			Data:    "",
		},
		callback:         "",
		wantBody:         `{"Code":0,"Message":"","Data":""}`,
		wantPanicPattern: "",
	})

	testOne("ok", args{
		response: &webapi.ApiResponse[any]{
			Code:    0,
			Message: "",
			Data:    map[string]int{"a": 1, "b": 2},
		},
		callback:         "",
		wantBody:         `{"Code":0,"Message":"","Data":{"a":1,"b":2}}`,
		wantPanicPattern: "",
	})

	testOne("callback", args{
		response: &webapi.ApiResponse[any]{
			Code:    0,
			Message: "",
			Data:    "",
		},
		callback:         "cb_name",
		wantBody:         `cb_name({"Code":0,"Message":"","Data":""})`,
		wantPanicPattern: "",
	})

	testOne("panic-no-response", args{
		wantPanicPattern: "Response not initialized",
	})

	testOne("panic-json-marshal", args{
		response: &webapi.ApiResponse[any]{
			Data: make(chan int),
		},
		wantPanicPattern: "json encoding error",
	})
}
