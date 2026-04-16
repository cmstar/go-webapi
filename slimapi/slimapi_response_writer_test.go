package slimapi

import (
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_slimApiResponseWriter_WriteResponse(t *testing.T) {
	type args struct {
		callData         any
		callback         string
		wantBody         []string
		wantPanicPattern string
	}
	testOne := func(a args) {
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
		state.Data = a.callData
		state.Handler = &webapi.ApiHandlerWrapper{
			ApiResponseBuilder: webapi.NewBasicApiResponseBuilder(),
		}

		if a.callback != "" {
			setCallback(state, a.callback)
		}
		NewSlimApiResponseWriter().WriteResponse(state)

		var body []string
		state.ResponseBody(func(block []byte) bool {
			body = append(body, string(block))
			return true
		})
		assert.Equal(t, a.wantBody, body)
	}

	t.Run("Empty", func(t *testing.T) {
		testOne(args{
			callData:         "",
			callback:         "",
			wantBody:         []string{`{"Code":0,"Message":"","Data":""}`},
			wantPanicPattern: "",
		})
	})

	t.Run("OK", func(t *testing.T) {
		testOne(args{
			callData:         map[string]int{"a": 1, "b": 2},
			callback:         "",
			wantBody:         []string{`{"Code":0,"Message":"","Data":{"a":1,"b":2}}`},
			wantPanicPattern: "",
		})
	})

	t.Run("Callback", func(t *testing.T) {
		testOne(args{
			callData:         map[string]int{"a": 1, "b": 2},
			callback:         "cb_name",
			wantBody:         []string{`cb_name({"Code":0,"Message":"","Data":{"a":1,"b":2}})`},
			wantPanicPattern: "",
		})
	})

	t.Run("Panic", func(t *testing.T) {
		testOne(args{
			callData:         make(chan int),
			wantPanicPattern: "json encoding error",
		})
	})

	t.Run("EventStream", func(t *testing.T) {
		testOne(args{
			callData: webapi.EventStream[int](func(yield func(data int, err error) bool) {
				for _, v := range []int{1, 2, 3} {
					if !yield(v, nil) {
						return
					}
				}
			}),
			wantBody: []string{
				`data: {"Code":0,"Message":"","Data":1}` + "\n\n",
				`data: {"Code":0,"Message":"","Data":2}` + "\n\n",
				`data: {"Code":0,"Message":"","Data":3}` + "\n\n",
				`event: END` + "\n" + `data: {"Code":1000,"Message":"","Data":null}` + "\n\n",
			},
			wantPanicPattern: "",
		})
	})

	t.Run("NdJson", func(t *testing.T) {
		testOne(args{
			callData: webapi.NdJson[string](func(yield func(data string, err error) bool) {
				for _, v := range []string{"a", "b", "c"} {
					if !yield(v, nil) {
						return
					}
				}
			}),
			wantBody: []string{
				`{"Code":0,"Message":"","Data":"a"}` + "\n",
				`{"Code":0,"Message":"","Data":"b"}` + "\n",
				`{"Code":0,"Message":"","Data":"c"}` + "\n",
				"",
			},
			wantPanicPattern: "",
		})
	})
}
