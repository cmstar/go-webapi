package slimapi

import (
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_slimApiNameResolver_parseMixedMetaParams(t *testing.T) {
	type valueCheck struct{ value, want string }
	type methodCheck valueCheck
	type formatCheck valueCheck
	type callbackCheck valueCheck

	resolver := &slimApiNameResolver{}
	test := func(input string, method methodCheck, format formatCheck, callback callbackCheck) {
		t.Run(input, func(t *testing.T) {
			resolver.parseMixedMetaParams(input, &method.value, &format.value, &callback.value)

			assert.Equal(t, method.want, method.value)
			assert.Equal(t, format.want, format.value)
			assert.Equal(t, callback.want, callback.value)
		})
	}

	test("mm~", methodCheck{"", "mm~"}, formatCheck{"", ""}, callbackCheck{"cb", "cb"})
	test("mm", methodCheck{"old", "old"}, formatCheck{"f", "f"}, callbackCheck{"cb", "cb"})
	test("mm.format", methodCheck{"", "mm"}, formatCheck{"", "format"}, callbackCheck{"", ""})
	test("mm(cb)", methodCheck{"", "mm"}, formatCheck{"", ""}, callbackCheck{"", "cb"})
	test("mm.fmt(cb)", methodCheck{"", "mm"}, formatCheck{"", "fmt"}, callbackCheck{"", "cb"})
	test("mm.fmt(cb)", methodCheck{"m", "m"}, formatCheck{"f", "f"}, callbackCheck{"c", "c"}) // 全部维持原值。

	// Broken values.
	test("mm.fmt(cb", methodCheck{"", "mm"}, formatCheck{"", "fmt"}, callbackCheck{"", ""})
	test("mm.((", methodCheck{"", "mm"}, formatCheck{"", ""}, callbackCheck{"", ""})
}

func Test_slimApiNameResolver_FillMethod(t *testing.T) {
	type want struct {
		name                string
		requestFormat       string
		responseContentType string
		callback            string
		routeParam          map[string]string
		errPattern          string // 校验 ApiState.Error 。
		panicPattern        string // 校验 panic 的消息。
	}

	testOne := func(relativeUrl string, requestContentType string, want want) {
		t.Run(relativeUrl, func(t *testing.T) {
			url := "http://temp.org/" + relativeUrl
			state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, url, webapitest.NewStateSetup{
				ContentType: string(requestContentType),
				RouteParams: want.routeParam,
			})

			if want.panicPattern != "" {
				defer func() {
					recovered := recover()
					require.NotNil(t, recovered)

					err, ok := recovered.(error)
					require.True(t, ok, "must be an error, got %T", recovered)

					assert.Regexp(t, want.panicPattern, err.Error())
				}()
			}

			resolver := NewSlimApiNameResolver()
			resolver.FillMethod(state)

			assert.Equal(t, want.name, state.Name)
			assert.Equal(t, want.responseContentType, state.ResponseContentType)

			requestFormat := getRequestFormat(state)
			assert.Equal(t, want.requestFormat, requestFormat)

			callback := getCallback(state)
			assert.Equal(t, want.callback, callback)

			if want.errPattern != "" {
				assert.NotNil(t, state.Error)
				assert.Regexp(t, want.errPattern, state.Error.Error(), "state.Error")
			}
		})
	}

	// 格式1：?~method=METHOD[&~format=FORMAT][&~callback=CALLBACK]
	testOne("?~method=name", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	testOne("?~method=name&~format=post", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Post,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	testOne("?~method=name&~format=plain", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypePlainText,
		callback:            "",
	})

	testOne("?~method=name&~format=post,plain", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Post,
		responseContentType: webapi.ContentTypePlainText,
		callback:            "",
	})

	// 通过 Content-Type 头指定格式。
	testOne("?~method=name", webapi.ContentTypeForm, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Post,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	// Content-Type 分号后的部分被忽略。
	testOne("?~method=name&multi=1", webapi.ContentTypeMultipartForm+"; boundary=----123", want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Post,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	// Content-Type 分后后的部分被忽略。
	testOne("?~method=name&with_charset=1", webapi.ContentTypeForm+"; charset=utf8", want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Post,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	// ~format 优先级比 Content-Type 高。
	testOne("?~method=name&~format=json", webapi.ContentTypeForm, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Json,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	// 带回调。
	testOne("?~method=name&~format=get&~callback=cb", webapi.ContentTypeForm, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	// JSONP 不受 plain 格式影响。
	testOne("?~method=name&~format=plain&~callback=cb", webapi.ContentTypeForm, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	// 格式2：?METHOD[.FORMAT][(CALLBACK)]
	testOne("?name&a=1", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	testOne("?name.json", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Json,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
	})

	testOne("?name.json(cb)", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Json,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	testOne("?name.plain(cb)", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	testOne("?name(cb)", webapi.ContentTypeNone, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	testOne("?name(cb)", webapi.ContentTypeJson, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Json,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
	})

	// 格式3基于路由参数。
	testOne("?a=1&b=2", webapi.ContentTypeJson, want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJavascript,
		callback:            "cb",
		routeParam: map[string]string{
			meta_Param_Method:   "name",
			meta_Param_Format:   "plain", // 指定 ~callback 时被忽略。
			meta_Param_Callback: "cb",
		},
	})

	testOne("no-query", webapi.ContentTypeJson+" ; charset=utf8", want{
		name:                "name",
		requestFormat:       meta_RequestFormat_Json,
		responseContentType: webapi.ContentTypePlainText,
		callback:            "",
		routeParam: map[string]string{
			meta_Param_Method: "name",
			meta_Param_Format: "plain,json",
		},
	})

	// 测试优先级，按格式 1,2,3 的顺序。
	testOne("?name2&~method=name1", webapi.ContentTypeNone, want{
		name:                "name1",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypeJson,
		callback:            "",
		routeParam: map[string]string{
			meta_Param_Method: "name3",
		},
	})

	testOne("?name2.plain", webapi.ContentTypeNone, want{
		name:                "name2",
		requestFormat:       meta_RequestFormat_Get,
		responseContentType: webapi.ContentTypePlainText,
		callback:            "",
		routeParam: map[string]string{
			meta_Param_Method: "name3",
			meta_Param_Format: "json",
		},
	})

	// 异常情况。
	testOne("?name.bad", webapi.ContentTypeNone, want{
		errPattern: "bad format",
	})
}
