package slimapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_slimApiDecoder_Decode(t *testing.T) {
	p := slimApiDecoderTestProvider{t}

	p.testOne(testOneArgs{methodName: "Empty"})

	p.testOne(testOneArgs{
		methodName: "F1",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"i": 3, // 大小写不敏感。
		},
		expected: []any{
			struct{ I int }{3},
		},
	})

	p.testOne(testOneArgs{
		methodName: "F3",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"i":           123,
			"stringField": "ss",
			"F":           "1.5",
		},
		expected: []any{
			simpleIn{123, "ss", float32(1.5), 0},
		},
	})

	p.testOne(testOneArgs{
		methodName: "F3",
		tag:        "mix",
		runMethods: RUN_GETPOST,
		requestQuery: map[string]any{
			"StringField": "part1", // 大小写不敏感，两个 StringField 会被合并。
			"noUse":       3,
		},
		requestBody: map[string]any{
			"i":           123,
			"stringfield": "part2",
			"F":           "1.5",
		},
		expected: []any{
			simpleIn{123, "part1,part2", float32(1.5), 0},
		},
	})

	p.testOne(testOneArgs{
		methodName: "F3",
		tag:        "override",
		runMethods: RUN_JSON,
		requestQuery: map[string]any{
			"stringfield": "override", // JSON 格式下，被覆盖。
			"noUse":       3,
		},
		requestBody: map[string]any{
			"i":           123,
			"stringField": "ss",
			"F":           "1.5",
		},
		expected: []any{
			simpleIn{123, "ss", float32(1.5), 0},
		},
	})

	valString := "val"
	p.testOne(testOneArgs{
		methodName: "Ptr",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"PTR": valString,
		},
		expected: []any{
			struct{ Ptr *string }{&valString},
		},
	})

	p.testOne(testOneArgs{
		methodName: "Slice",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"Sl": "11~22~33",
		},
		expected: []any{
			struct{ Sl []uint64 }{[]uint64{11, 22, 33}},
		},
	})

	p.testOne(testOneArgs{
		methodName: "Time",
		tag:        "short",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"Std": "2022-04-17 21:18:25",
			"Loc": "2022-11-03 07:30:06",
		},
		expected: []any{
			timeIn{
				Std: time.Date(2022, 4, 17, 21, 18, 25, 0, time.UTC),
				Loc: Time(time.Date(2022, 11, 03, 7, 30, 6, 0, time.UTC)),
			},
		},
	})

	p.testOne(testOneArgs{
		methodName: "Time",
		tag:        "long",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"Std": "2022-04-17 21:18:25.12345",
			"Loc": "2022-11-03 07:30:06.321",
		},
		expected: []any{
			timeIn{
				Std: time.Date(2022, 4, 17, 21, 18, 25, int(123450*time.Microsecond), time.UTC),
				Loc: Time(time.Date(2022, 11, 03, 7, 30, 6, int(321*time.Millisecond), time.UTC)),
			},
		},
	})

	p.testOne(testOneArgs{
		methodName: "Time",
		tag:        "RFC3339",
		runMethods: RUN_ALL,
		requestBody: map[string]any{
			"Std": "2022-04-17T21:18:25.12345Z",
			"Loc": "2022-11-03T07:30:06.321Z",
		},
		expected: []any{
			timeIn{
				Std: time.Date(2022, 4, 17, 21, 18, 25, int(123450*time.Microsecond), time.UTC),
				Loc: Time(time.Date(2022, 11, 03, 7, 30, 6, int(321*time.Millisecond), time.UTC)),
			},
		},
	})

	p.testOne(testOneArgs{
		methodName: "Complex",
		runMethods: RUN_JSON, // 嵌套复杂类型不支持 GET 。
		requestBody: map[string]any{
			"F3Slice": []any{
				map[string]any{"I": 12},
				map[string]any{"StringField": "gg"},
			},
			"MM": map[string]any{
				"k1": []int{3, 2, 1},
				"k2": "11~22~33",
			},
			"Boolean": true,
		},
		expected: []any{
			complexIn{
				F3Slice: []*simpleIn{
					{I: 12},
					{StringField: "gg"},
				},
				MM: map[string][]int{
					"k1": {3, 2, 1},
					"k2": {11, 22, 33},
				},
				Boolean: true,
			},
		},
	})

	p.testOne(testOneArgs{
		methodName: "WithApiState",
		runMethods: RUN_ALL,
		expected: []any{
			EXPECT_API_STATE,
		},
	})

	p.testOne(testOneArgs{
		methodName: "WithAll",
		runMethods: RUN_ALL,
		expected: []any{
			EXPECT_API_STATE,
			simpleIn{},
		},
	})

	p.testOne(testOneArgs{
		methodName: "WithAllReverse",
		runMethods: RUN_ALL,
		expected: []any{
			simpleIn{},
			EXPECT_API_STATE,
		},
	})

	p.testOne(testOneArgs{
		methodName:      "TooManyParameters1",
		runMethods:      RUN_GET,
		panicMsgPattern: `argument type cannot be duplicated`,
	})

	p.testOne(testOneArgs{
		methodName:      "TooManyParameters2",
		runMethods:      RUN_GET,
		panicMsgPattern: `argument type cannot be duplicated`,
	})

	p.testOne(testOneArgs{
		methodName:      "WrongTypeParameters",
		runMethods:      RUN_GET,
		panicMsgPattern: `method '' arg0 chan string: not supported`,
	})

	p.testOne(testOneArgs{
		methodName:      "WrongTypeParameters2",
		runMethods:      RUN_GET,
		panicMsgPattern: `method '' arg0 webapi.ApiState: must be a pointer`,
	})

	p.testOne(testOneArgs{
		methodName: "CannotConvert",
		runMethods: RUN_GET,
		requestBody: map[string]any{
			"C": 1,
		},
		errPattern:      "bad request",
		panicMsgPattern: ``,
	})
}

const urlBase = "http://temp.org/path/"

// 用来封装测试需要的方法，公开方法作为 ApiState.Method ，非公开方法则是辅助方法。
type slimApiDecoderTestProvider struct {
	t *testing.T
}

/*
 * 作为 ApiState.Method 的方法。
 */

func (slimApiDecoderTestProvider) Empty()             {}
func (slimApiDecoderTestProvider) F1(struct{ I int }) {}

type simpleIn struct {
	I           int
	StringField string
	F           float32
	Ignored     int
}

type timeIn struct {
	Std time.Time
	Loc Time
}

func (slimApiDecoderTestProvider) F3(simpleIn)                 {}
func (slimApiDecoderTestProvider) Ptr(struct{ Ptr *string })   {}
func (slimApiDecoderTestProvider) Slice(struct{ Sl []uint64 }) {}
func (slimApiDecoderTestProvider) Time(timeIn)                 {}

type complexIn struct {
	F3Slice []*simpleIn
	MM      map[string][]int
	Boolean bool
}

func (slimApiDecoderTestProvider) Complex(complexIn)                         {}
func (slimApiDecoderTestProvider) WithApiState(*webapi.ApiState)             {}
func (slimApiDecoderTestProvider) WithAll(*webapi.ApiState, simpleIn)        {}
func (slimApiDecoderTestProvider) WithAllReverse(simpleIn, *webapi.ApiState) {}

// These should panic.
func (slimApiDecoderTestProvider) TooManyParameters1(complexIn, complexIn)                   {}
func (slimApiDecoderTestProvider) TooManyParameters2(*webapi.ApiState, complexIn, complexIn) {}
func (slimApiDecoderTestProvider) WrongTypeParameters(chan string)                           {}
func (slimApiDecoderTestProvider) WrongTypeParameters2(webapi.ApiState)                      {}
func (slimApiDecoderTestProvider) CannotConvert(struct{ C chan int })                        {}

/*
 * 下面是用于执行测试流程的方法和类型。
 */

// 通过一个位标记，指定要执行哪些请求类型的测试。
type RunRequestType uint

const (
	RUN_GET            RunRequestType = 1 << iota // 执行 GET 请求的测试。
	RUN_POST_QUERY                                // 指定表单 POST 的测试。
	RUN_JSON                                      // 执行 JSON POST 的测试。
	RUN_MULTIPART_FORM                            // 执行 multipart/form-data 型表单的测试。

	RUN_GETPOST = RUN_GET | RUN_POST_QUERY | RUN_MULTIPART_FORM
	RUN_ALL     = RUN_GETPOST | RUN_JSON
)

// 用于在 testOneArgs.expected 里代替特定的值，这些值构造比较复杂，难以直接校验，校验时匹配类型即可。
type ExpectedSpecialType string

const (
	EXPECT_API_STATE ExpectedSpecialType = "<webapi.ApiState>" // 参数是 webapi.ApiState 。
)

type testOneArgs struct {
	methodName      string         // 方法名称。
	tag             string         // 用于备注测试用例。
	runMethods      RunRequestType // 位标记，指定要执行的请求类型。
	requestQuery    map[string]any // 固定放在 URL 上的输入参数。通过 HTTP 请求发送，反序列化后传给 methodName 对应方法。
	requestBody     map[string]any // Body 部分的输入参数。 GET 时会和 requestQuery 合并在一起。
	expected        []any          // 预期的解析结果，顺序需和 methodName 对应方法的入参一致。可以用 ExpectedSpecialType 指代特定类型。
	errPattern      string         // 断言 ApiState.Error 的消息。
	panicMsgPattern string         // 正则，用于验证 panic 的消息；若预期不会 panic ，则为空。
}

// 测试一个方法。
func (p slimApiDecoderTestProvider) testOne(args testOneArgs) {
	checkRecoveredError := func(t *testing.T, recovered any) {
		require.NotNil(t, recovered, "should panic")
		apiErr, ok := recovered.(webapi.ApiError)
		require.Truef(t, ok, "must panic ApiError, got %T: %v", recovered, recovered)
		require.Regexp(t, args.panicMsgPattern, apiErr.Error())
	}

	// map，和 slice 可以是 nil ，影响序列化和结果比对，统一转成空集。
	expected := args.expected
	if args.expected == nil {
		expected = make([]any, 0)
	}

	requestQuery := args.requestQuery
	if requestQuery == nil {
		requestQuery = make(map[string]any)
	}

	requestBody := args.requestBody
	if requestBody == nil {
		requestBody = make(map[string]any)
	}

	buildTestName := func(runMethod string) string {
		name := args.methodName + "_" + runMethod
		if args.tag != "" {
			name += "_" + args.tag
		}
		return name
	}

	if args.runMethods == 0 || args.runMethods&RUN_GET == RUN_GET {
		p.t.Run(buildTestName("GET"), func(t *testing.T) {
			if args.panicMsgPattern != "" {
				defer func() { checkRecoveredError(t, recover()) }()
			}
			p.doTestGet(args.methodName, requestQuery, requestBody, expected, args.errPattern)
		})
	}

	if args.runMethods == 0 || args.runMethods&RUN_POST_QUERY == RUN_POST_QUERY {
		p.t.Run(buildTestName("POST"), func(t *testing.T) {
			if args.panicMsgPattern != "" {
				defer func() { checkRecoveredError(t, recover()) }()
			}
			p.doTestPostForm(args.methodName, requestQuery, requestBody, expected, args.errPattern)
		})
	}

	if args.runMethods == 0 || args.runMethods&RUN_JSON == RUN_JSON {
		p.t.Run(buildTestName("JSON"), func(t *testing.T) {
			if args.panicMsgPattern != "" {
				defer func() { checkRecoveredError(t, recover()) }()
			}
			p.doTestPostJson(args.methodName, requestQuery, requestBody, expected, args.errPattern)
		})
	}

	if args.runMethods == 0 || args.runMethods&RUN_MULTIPART_FORM == RUN_MULTIPART_FORM {
		p.t.Run(buildTestName("MULTIPART"), func(t *testing.T) {
			if args.panicMsgPattern != "" {
				defer func() { checkRecoveredError(t, recover()) }()
			}
			p.doTestMultipartForm(args.methodName, requestQuery, requestBody, expected, args.errPattern)
		})
	}
}

func (p slimApiDecoderTestProvider) doTestGet(
	methodName string,
	requestQuery map[string]any,
	requestBody map[string]any, // Merge with requestQuery.
	expected []any,
	errPattern string,
) {
	url := urlBase
	marked := false
	if len(requestQuery) > 0 {
		url += "?" + p.buildQueryString(requestQuery)
		marked = true
	}

	if len(requestBody) > 0 {
		if marked {
			url += "&"
		} else {
			url += "?"
		}
		url += p.buildQueryString(requestBody)
	}

	state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, url, webapitest.NewStateSetup{})
	p.doTestDecode(state, methodName, meta_RequestFormat_Get, expected, errPattern)
}

func (p slimApiDecoderTestProvider) doTestPostForm(
	methodName string,
	requestQuery map[string]any,
	requestBody map[string]any,
	expected []any,
	errPattern string,
) {
	url := urlBase
	if len(requestQuery) > 0 {
		url += "?" + p.buildQueryString(requestQuery)
	}

	body := p.buildQueryString(requestBody)
	state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, url, webapitest.NewStateSetup{
		HttpMethod:  http.MethodPost,
		ContentType: webapi.ContentTypeForm,
		BodyString:  body,
	})
	p.doTestDecode(state, methodName, meta_RequestFormat_Post, expected, errPattern)
}

func (p slimApiDecoderTestProvider) doTestPostJson(
	methodName string,
	requestQuery map[string]any,
	requestBody map[string]any,
	expected []any,
	errPattern string,
) {
	url := urlBase
	if len(requestQuery) > 0 {
		url += "?" + p.buildQueryString(requestQuery)
	}

	jsonBytes, err := json.Marshal(requestBody)
	require.NoError(p.t, err, "to json")

	state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, url, webapitest.NewStateSetup{
		HttpMethod:  http.MethodPost,
		ContentType: webapi.ContentTypeJson,
		BodyReader:  bytes.NewBuffer(jsonBytes),
	})
	p.doTestDecode(state, methodName, meta_RequestFormat_Json, expected, errPattern)
}

func (p slimApiDecoderTestProvider) doTestMultipartForm(
	methodName string,
	requestQuery map[string]any,
	requestBody map[string]any,
	expected []any,
	errPattern string,
) {
	url := urlBase
	if len(requestQuery) > 0 {
		url += "?" + p.buildQueryString(requestQuery)
	}

	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	for k, v := range requestBody {
		w.WriteField(k, fmt.Sprintf("%v", v))
	}
	err := w.Close()
	if err != nil {
		require.NoError(p.t, err)
	}

	// 为便于调试，多消耗点资源，将 body 放到字符串里。
	bodyBytes, _ := io.ReadAll(buf)
	bodyString := string(bodyBytes)

	state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, url, webapitest.NewStateSetup{
		HttpMethod:  http.MethodPost,
		ContentType: w.FormDataContentType(),
		BodyReader:  strings.NewReader(bodyString),
	})
	p.doTestDecode(state, methodName, meta_RequestFormat_Post, expected, errPattern)
}

func (slimApiDecoderTestProvider) buildQueryString(nameValues map[string]any) string {
	res := ""
	for name, value := range nameValues {
		if len(res) > 0 {
			res += "&"
		}
		res += name + "=" + url.QueryEscape(fmt.Sprintf("%v", value))
	}
	return res
}

func (p slimApiDecoderTestProvider) doTestDecode(
	state *webapi.ApiState, methodName string, format string, expected []any, errPattern string) {
	setRequestFormat(state, format)
	state.Method = webapi.ApiMethod{
		Name:     methodName,
		Value:    reflect.ValueOf(p).MethodByName(methodName),
		Provider: "",
	}

	decoder := NewSlimApiDecoder()
	decoder.Decode(state)

	iArgs := make([]any, 0, len(state.Args))
	for i := 0; i < len(state.Args); i++ {
		// 对于不好检测的类型，转成对应的常量进行比对。
		switch v := state.Args[i].Interface().(type) {
		case *webapi.ApiState:
			iArgs = append(iArgs, EXPECT_API_STATE)
		default:
			iArgs = append(iArgs, v)
		}
	}
	assert.Equal(p.t, expected, iArgs)

	if errPattern != "" {
		assert.NotNil(p.t, state.Error, "state.Error")
		assert.Regexp(p.t, errPattern, state.Error.Error(), "state.Error")
	}
}
