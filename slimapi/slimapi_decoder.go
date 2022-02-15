package slimapi

import (
	"encoding/json"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-webapi"
)

// slimApiDecoder 实现 SlimAPI 的 webapi.ApiDecoder 。
type slimApiDecoder struct {
}

// NewSlimApiDecoder 返回用于 SlimAPI 协议的 webapi.ApiDecoder 实现。
func NewSlimApiDecoder() webapi.ApiDecoder {
	return &slimApiDecoder{}
}

// Decode 实现 webapi.ApiDecoder.Decode 。
func (d *slimApiDecoder) Decode(state *webapi.ApiState) {
	state.MustHaveMethod()

	typ := state.Method.Value.Type()
	numIn := typ.NumIn()
	args := make([]reflect.Value, numIn)

	if numIn == 0 {
		state.Args = args
		return
	}

	paramHit := false
	for i := 0; i < numIn; i++ {
		in := typ.In(i)
		switch {
		case in == reflect.TypeOf(state):
			args[i] = reflect.ValueOf(state)

		case in.Kind() == reflect.Struct && in != reflect.TypeOf(*state):
			// 自定义参数只能有一个。
			if paramHit {
				webapi.PanicApiError(state, nil, "there shouldn't be more than one parameter for the custom data")
			}

			paramMap, err := d.paramMap(state)
			if err != nil {
				state.Error = webapi.CreateBadRequestError(state, err, "bad request")
				return
			}

			val, err := slimApiConv.ConvertType(paramMap, in)
			if err != nil {
				state.Error = webapi.CreateBadRequestError(state, err, "bad request")
				return
			}

			args[i] = reflect.ValueOf(val)
			paramHit = true

		default:
			webapi.PanicApiError(state, nil, "the parameter must be one of *webapi.ApiState or a struct, got %v", in)
		}
	}

	state.Args = args
}

// paramMap 将各类参数存入 map[string]interface{} 。
func (d *slimApiDecoder) paramMap(state *webapi.ApiState) (map[string]interface{}, error) {
	format := getRequestFormat(state)
	if format == "" {
		webapi.PanicApiError(state, nil, "missing request format")
	}

	switch format {
	case meta_RequestFormat_Get:
		// 注意用自己解析的这个 Query 。
		m := make(map[string]interface{})
		for k, v := range state.Query.Named {
			m[k] = v
		}
		return m, nil

	case meta_RequestFormat_Post:
		req := state.Ctx.Request()
		contentType := req.Header.Get(webapi.HttpHeaderContentType)

		// Content-Type 为 multipart/form-data 的，交给框架内置方法解析。
		// 为 application/x-www-form-urlencoded 或其他的 Content-Type 的，则单独读取，
		// 此时的值类似 URL 上的 query-string ，需要使用同样的规则处理。
		if strings.Index(contentType, webapi.ContentTypeMultipartForm) == 0 {
			return d.readMultiPartBody(state)
		}
		return d.readQueryStringBody(state, contentType), nil

	case meta_RequestFormat_Json:
		return d.readJsonBody(state)

	default:
		webapi.PanicApiError(state, nil, "unsupported format: %v", format)
	}

	return nil, nil // never run
}

func (d *slimApiDecoder) readQueryStringBody(state *webapi.ApiState, contentType string) map[string]interface{} {
	// 将整个 body 作为 query-string 读取。不知道 body 实际上会上送什么样的数据，做一层防御，限制读取数据的最大大小。
	reader := io.LimitReader(state.Ctx.Request().Body, maxMemorySizeParseRequestBody)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, reader)
	if err != nil {
		// 这里一般不会出错。若出错了就比较严重了，直接 panic 。
		webapi.PanicApiError(state, err, "error on reading the '%s' body", contentType)
	}

	form := buf.String()
	query := webapi.ParseQueryString(form)
	m := make(map[string]interface{})
	for k, v := range query.Named {
		m[k] = v
	}

	setBufferedBody(state, form)
	return m
}

func (d *slimApiDecoder) readMultiPartBody(state *webapi.ApiState) (map[string]interface{}, error) {
	req := state.Ctx.Request()
	err := req.ParseMultipartForm(maxMemorySizeParseRequestBody)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: read multipart-form", err)
		return nil, err
	}

	buf := new(strings.Builder)
	m := make(map[string]interface{})
	for k, vs := range req.Form {
		v := strings.Join(vs, ",")
		m[k] = v

		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
	}

	body := buf.String()
	setBufferedBody(state, body)
	return m, nil
}

func (d *slimApiDecoder) readJsonBody(state *webapi.ApiState) (map[string]interface{}, error) {
	body, err := io.ReadAll(state.Ctx.Request().Body)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: read body", err)
		return nil, err
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(body, &m)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: json unmarshal", err)
		return nil, err
	}

	// json.Unmarshal 接收 []byte 而这里接收 string ，转换有点开销，但目前没啥好方案解决。
	setBufferedBody(state, string(body))
	return m, nil
}
