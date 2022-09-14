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

// NewSlimApiDecoder 返回用于 SlimAPI 协议的 [webapi.ApiDecoder] 实现。
func NewSlimApiDecoder() webapi.ArgumentDecoderPipeline {
	return webapi.NewArgumentDecoderPipeline(StructArgumentDecoder)
}

// StructArgumentDecoder 是一个 [webapi.ArgumentDecoder] ，
// 定义了 SlimAPI 协议的参数解析过程，用于方法参数表中 struct 类型的参数。
//
// 这是一个单例。
var StructArgumentDecoder = slimApiMethodStructArgDecoder{}

type slimApiMethodStructArgDecoder struct{}

// DecodeArg implements [webapi.ApiDecoder.DecodeArg].
func (d slimApiMethodStructArgDecoder) DecodeArg(state *webapi.ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
	if argType.Kind() != reflect.Struct {
		return false, nil, nil
	}

	paramMap, err := d.paramMap(state)
	if err != nil {
		return false, nil, webapi.CreateBadRequestError(state, err, "bad request")
	}

	val, err := Conv.ConvertType(paramMap, argType)
	if err != nil {
		return false, nil, webapi.CreateBadRequestError(state, err, "bad request")
	}

	return true, val, nil
}

// paramMap 将各类参数存入 map[string]any 。
//  1. 参数是大小写不敏感的。
//  2. URL 上的参数（query）总是会被读取。
//  3. 表单参数会与 query 合并在一起，同名（大小写不敏感）参数的值会被用逗号拼接起来。
//  4. JSON 参数会与 query 合并在一起，同名的参数， JSON 的值会将 query 的值覆盖掉。
func (d slimApiMethodStructArgDecoder) paramMap(state *webapi.ApiState) (map[string]any, error) {
	format := getRequestFormat(state)
	if format == "" {
		webapi.PanicApiError(state, nil, "missing request format")
	}

	switch format {
	case meta_RequestFormat_Get:
		m := d.readQueryInLowercase(state)
		return m, nil

	case meta_RequestFormat_Post:
		req := state.RawRequest
		contentType := req.Header.Get(webapi.HttpHeaderContentType)

		// Content-Type 为 multipart/form-data 的，交给框架内置方法解析。
		// 为 application/x-www-form-urlencoded 或其他的 Content-Type 的，则单独读取，
		// 此时的值类似 URL 上的 query-string ，需要使用同样的规则处理。
		if strings.Index(contentType, webapi.ContentTypeMultipartForm) == 0 {
			return d.readMultiPartForm(state)
		}

		m := d.readForm(state, contentType)
		return m, nil

	case meta_RequestFormat_Json:
		return d.readJsonBody(state)

	default:
		webapi.PanicApiError(state, nil, "unsupported format: %v", format)
	}

	return nil, nil // never run
}

// 读取 URL 上的参数，返回的参数名称总是小写的，值总是 string 。
func (d slimApiMethodStructArgDecoder) readQueryInLowercase(state *webapi.ApiState) map[string]any {
	// 用自己解析的这个 Query 。
	m := make(map[string]any)
	for k, v := range state.Query.Named {
		m[k] = v
	}
	return m
}

func (d slimApiMethodStructArgDecoder) readForm(state *webapi.ApiState, contentType string) map[string]any {
	// 将整个 body 作为 query-string 读取。不知道 body 实际上会上送什么样的数据，做一层防御，限制读取数据的最大大小。
	reader := io.LimitReader(state.RawRequest.Body, maxMemorySizeParseRequestBody)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, reader)
	if err != nil {
		// 这里一般不会出错。若出错了就比较严重了，直接 panic 。
		webapi.PanicApiError(state, err, "error on reading the '%s' body", contentType)
	}

	lowercaseParams := d.readQueryInLowercase(state)
	form := buf.String()
	query := webapi.ParseQueryString(form)
	for k, v := range query.Named {
		k = strings.ToLower(k)
		old, ok := lowercaseParams[k]
		if ok {
			lowercaseParams[k] = old.(string) + "," + v
		} else {
			lowercaseParams[k] = v
		}
	}

	setBufferedBody(state, form)
	return lowercaseParams
}

func (d slimApiMethodStructArgDecoder) readMultiPartForm(state *webapi.ApiState) (map[string]any, error) {
	req := state.RawRequest

	// ParseMultipartForm 会将 URL 和 body 上的参数都合并到 req.Form 上。
	err := req.ParseMultipartForm(maxMemorySizeParseRequestBody)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: read multipart-form", err)
		return nil, err
	}

	lowercaseParams := d.readQueryInLowercase(state)
	buf := new(strings.Builder)
	for k, vs := range req.PostForm {
		k = strings.ToLower(k)
		v := strings.Join(vs, ",")

		// Form 里的参数是区分大小写的，需要以大小写不敏感的方式将它们并起来。
		old, ok := lowercaseParams[k]
		if ok {
			lowercaseParams[k] = old.(string) + "," + v
		} else {
			lowercaseParams[k] = v
		}

		if buf.Len() > 0 {
			buf.WriteRune('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
	}

	body := buf.String()
	setBufferedBody(state, body)
	return lowercaseParams, nil
}

func (d slimApiMethodStructArgDecoder) readJsonBody(state *webapi.ApiState) (map[string]any, error) {
	body, err := io.ReadAll(state.RawRequest.Body)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: read body", err)
		return nil, err
	}

	lowercaseParam := d.readQueryInLowercase(state)
	fromBody := make(map[string]any)
	err = json.Unmarshal(body, &fromBody)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: json unmarshal", err)
		return nil, err
	}

	// json.Unmarshal 接收 []byte 而这里接收 string ，转换有点开销，但目前没啥好方案解决。
	setBufferedBody(state, string(body))

	for k, v := range fromBody {
		// 采用先删再加的方式，使 JSON 字段尽量维持原来的样子。
		delete(lowercaseParam, strings.ToLower(k))
		lowercaseParam[k] = v
	}
	return lowercaseParam, nil
}
