package slimapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationTestArgs struct {
	requestRelativeUrl string // 不以 / 开头。
	requestContentType string
	requestBody        string
	requestRouteParam  map[string]string
	wantStatusCode     int
	wantContentType    string
	wantBody           string
	wantLogPattern     map[string]string // 可以对每个字段单独用正则匹配。
}

var handlerForIntegrationTest webapi.ApiHandler

// 承载 API 方法。便于维护，每个 API 方法和对应的测试过程写在一起。
type integrationTestMethodProvider struct{}

func init() {
	handlerForIntegrationTest = NewSlimApiHandler("")
	handlerForIntegrationTest.RegisterMethods(integrationTestMethodProvider{})
}

// 是否向控制台输出捕获到的日志。
const enableLogOutput = false

// 跑一个集成测试用例，测试 HTTP 请求的输入输出，不测试中间过程。
func DoIntegrationTest(t *testing.T, args integrationTestArgs) {
	var httpMethod string
	switch args.requestContentType {
	case webapi.ContentTypeForm:
		fallthrough
	case webapi.ContentTypeJson:
		httpMethod = "POST"
	default:
		httpMethod = "GET"
	}

	url := "http://temp.org/" + args.requestRelativeUrl
	state, rec := webapitest.NewStateForTest(handlerForIntegrationTest, url, webapitest.NewStateSetup{
		HttpMethod:  httpMethod,
		ContentType: string(args.requestContentType),
		RouteParams: args.requestRouteParam,
		BodyString:  args.requestBody,
	})

	logger := webapitest.NewLogRecorder()
	logFinder := logx.NewSingleLoggerLogFinder(logger)
	handlerFunc := webapi.CreateHandlerFunc(handlerForIntegrationTest, logFinder)
	handlerFunc(state.RawResponse, state.RawRequest)

	a := assert.New(t)
	a.Equal(args.wantStatusCode, rec.Code, "StatusCode")
	a.Equal(args.wantContentType, rec.Header()[webapi.HttpHeaderContentType][0], "ContentType")
	a.Equal(args.wantBody, rec.Body.String(), "Body")

	logMap := logger.Map()
	require.NotNil(t, logMap, "m")
	require.Equal(t, 1, len(logMap), "len(m)")

	if args.wantLogPattern != nil {
		for key, pattern := range args.wantLogPattern {
			v, ok := logger.Map()[0][key]
			if ok {
				a.Regexp(pattern, v, key)
			} else {
				a.Fail("missing log field " + key)
			}
		}
	}

	if enableLogOutput {
		logMsg := logger.String()
		fmt.Println("log>>>>>>>>>>")
		fmt.Println(logMsg)
		fmt.Println("<<<<<<<<<<")
	}
}

func TestSlimApi_Empty(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?Empty",
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":null}`,
	})
}

func TestSlimApi_Plus_get(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?&~method=plus&a=1&b=2",
		requestContentType: "",
		requestBody:        "a=3&b=4", // The body should be ignored.
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":3}`,
	})
}

func TestSlimApi_Plus_post(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?~format=post&~method=plus",
		requestContentType: "",
		requestBody:        "a=1&b=2",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":3}`,
	})
}

func TestSlimApi_Plus_json(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?plus",
		requestContentType: "application/json",
		requestBody:        `{"a":1,"b":"2"}`,
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":3}`,
	})
}

func TestSlimApi_Plus_panic(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?plus.post",
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":500,"Message":"internal error","Data":null}`,
		wantLogPattern: map[string]string{
			"ErrorType": "ErrorWrapper",
			"Error":     "nil pointer",
		},
	})
}

func TestSlimApi_SumAndShowMap_get_route(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?S=1~2~3",
		requestContentType: "",
		requestBody:        "S=100", // Ignored.
		requestRouteParam:  map[string]string{"~method": "SumAndShowMap"},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":{"Sum":6,"M":null}}`,
	})
}

func TestSlimApi_SumAndShowMap_json(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?SumAndShowMap.json",
		requestContentType: "",
		requestBody:        `{ "S":[1,3], "M":{"A":1} }`, // map 是乱序的，所以 M 只能放一个字段，不然序列化后顺序错乱不好检验。
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":{"Sum":4,"M":{"A":1}}}`,
	})
}

func TestSlimApi_SumAndShowMap_decodeError(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?SumAndShowMap.json",
		requestContentType: "",
		requestBody:        `{ "M":1 }`,
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":400,"Message":"bad request","Data":null}`,
		wantLogPattern: map[string]string{
			"ErrorType": "BadRequestError",
			"Error":     `bad request\n=== .+`,
		},
	})
}

func TestSlimApi_Time_json(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?Time",
		requestContentType: "application/json",
		requestBody:        `{ "T":"2022-01-15 18:22:59" }`,
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":{"T":"2022-01-15 18:22:59"}}`,
	})
}

func TestSlimApi_ShowError_noError(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?ShowError&S=gg",
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":0,"Message":"","Data":"gg"}`,
	})
}

func TestSlimApi_ShowError_stringError(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?ShowError&S=gg&E=msg&Type=" + ShowError_StringError,
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":500,"Message":"internal error","Data":"gg"}`,
		wantLogPattern: map[string]string{
			"ErrorType": "errorString",
			"Error":     "msg",
		},
	})
}

func TestSlimApi_ShowError_bizError(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?ShowError&S=gg&E=msg&Type=" + ShowError_BizError999,
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":999,"Message":"msg","Data":"gg"}`,
		wantLogPattern: map[string]string{
			"level":     "WARN",
			"ErrorType": "BizError",
			"Error":     `\(999\) msg`,
		},
	})
}

func TestSlimApi_ShowError_panicString(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?ShowError&S=gg&E=msg&Type=" + ShowError_PanicString,
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":500,"Message":"internal error","Data":null}`,
		wantLogPattern: map[string]string{
			"level":     "ERROR",
			"ErrorType": "ErrorWrapper",
			"Error":     `(?s)^ShowError: msg\n--- \[.+=== msg\n$`,
		},
	})
}

func TestSlimApi_ShowError_panicBizError(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?ShowError&S=gg&E=msg&Type=" + ShowError_PanicBizError999,
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":999,"Message":"msg","Data":null}`,
		wantLogPattern: map[string]string{
			"level":     "WARN",
			"ErrorType": "BizError",
			"Error":     `\(999\) msg`,
		},
	})
}

func TestSlimApi_CannotDecode(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?CannotDecode&c=",
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":400,"Message":"bad request","Data":null}`,
		wantLogPattern: map[string]string{
			"level":     "ERROR",
			"ErrorType": "BadRequestError",
			"Error":     `bad request\n=== conv`,
		},
	})
}

func TestSlimApi_CannotEncode(t *testing.T) {
	DoIntegrationTest(t, integrationTestArgs{
		requestRelativeUrl: "?CannotEncode",
		requestContentType: "",
		requestBody:        "",
		requestRouteParam:  map[string]string{},
		wantStatusCode:     200,
		wantContentType:    webapi.ContentTypeJson,
		wantBody:           `{"Code":500,"Message":"internal error","Data":null}`,
		wantLogPattern: map[string]string{
			"level":     "ERROR",
			"ErrorType": "ErrorWrapper",
			"Error":     `json encoding error`,
		},
	})
}

func (integrationTestMethodProvider) Empty() {}

type PlusRequest struct {
	A int
	B *int
}

func (integrationTestMethodProvider) Plus(args PlusRequest) *int {
	res := args.A + *args.B
	return &res
}

type SumAndShowMapRequest struct {
	S []float32
	M map[string]float32
}

type SumAndShowMapResponse struct {
	Sum float32
	M   map[string]float32
}

func (integrationTestMethodProvider) SumAndShowMap(args SumAndShowMapRequest) SumAndShowMapResponse {
	var sum float32
	for _, v := range args.S {
		sum += v
	}
	return SumAndShowMapResponse{
		Sum: sum,
		M:   args.M,
	}
}

func (integrationTestMethodProvider) Time(req struct{ T Time }) struct{ T Time } {
	req.T = Time(req.T.Time().UTC())
	return req
}

const (
	ShowError_NoError          = "0"
	ShowError_StringError      = "1"
	ShowError_PanicString      = "2"
	ShowError_BizError999      = "3"
	ShowError_PanicBizError999 = "4"
)

type ShowErrorRequest struct {
	Type string // 指定如何输出错误。
	E    string // 输出错误时作为错误描述。
	S    string
}

func (integrationTestMethodProvider) ShowError(args ShowErrorRequest) (s string, e error) {
	switch args.Type {
	case ShowError_StringError:
		return args.S, errors.New(args.E)
	case ShowError_PanicString:
		panic(errors.New(args.E))
	case ShowError_BizError999:
		return args.S, errx.NewBizError(999, args.E, nil)
	case ShowError_PanicBizError999:
		panic(errx.NewBizError(999, args.E, nil))
	default:
		return args.S, nil
	}
}

type CannotDecodeRequest struct {
	C chan int
}

func (integrationTestMethodProvider) CannotDecode(args CannotDecodeRequest) {
}

type CannotEncodeResponse struct {
	C chan int
}

func (integrationTestMethodProvider) CannotEncode() CannotEncodeResponse {
	return CannotEncodeResponse{}
}

func (integrationTestMethodProvider) Panic(v interface{}) {
	panic(v)
}
