package slimapi

import (
	"encoding/json"
	"io"
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
		if strings.HasPrefix(contentType, webapi.ContentTypeMultipartForm) {
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

// 读取 URL 上的参数，包含 query-string 和路由参数。
// 参数名称（ key ）是大小写不敏感的，总是被转换为小写，值（ value ）总是 string 。
// 如果一个参数同时出现在 query 和路由上，会使用 query 的值（实际使用就应避免这种情况）。
func (d slimApiMethodStructArgDecoder) readQueryInLowercase(state *webapi.ApiState) map[string]any {
	m := make(map[string]any)

	// 路由表。
	routeParams := webapi.AllRouteParams(state.RawRequest)
	for _, v := range routeParams {
		m[strings.ToLower(v.Key)] = v.Value
	}

	// 用自己解析的这个 Query 。
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

	setRequestBodyDescription(state, form)
	return lowercaseParams
}

// 解析 multipart/form-data 类型的请求。以下内容会被放在返回的 map 里：
//   - URL 上的参数（ query ）。
//   - body 中的 text/plain 类型的 part : Content-Disposition 的 name 作为 key ，内容作为 value ，类型为 string 。
//   - body 中的 application/json 类型的 part ： Content-Disposition 的 name 作为 key ，内容作为 value ，类型为 JSON 反序列化后的 map[string]any 。
//     此类型的 part 可用于解决上传文件的同事传递复杂结构参数的需求。
//
// 如果同名的 part 有多个，仅保留最后一个。
func (d slimApiMethodStructArgDecoder) readMultiPartForm(state *webapi.ApiState) (map[string]any, error) {
	req := state.RawRequest

	err := req.ParseMultipartForm(maxMemorySizeParseRequestBody)
	if err != nil {
		err = errx.Wrap("slimApiDecoder: parse multipart-form", err)
		return nil, err
	}

	// URL 上的参数（ query ）。
	lowercaseParams := d.readQueryInLowercase(state)

	// body 中的 text/plain 类型的 part 。
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
	}

	// body 中的文件类型的 part 。
	for name, fileHeader := range req.MultipartForm.File {
		// 同名文件只取最后一个。
		file := fileHeader[len(fileHeader)-1]

		// 转换成 *FilePart ，其受 Conv 对象的支持。
		filePart, err := NewFilePart(file)
		if err != nil {
			err = errx.Wrap("slimApiDecoder: parse multipart-form", err)
			return nil, err
		}
		lowercaseParams[strings.ToLower(name)] = filePart
	}

	setRequestBodyDescription(state, lowercaseParams)
	return lowercaseParams, nil
}

// 将整个 HTTP body 作为一个 JSON 处理。要求其必须是一个 JSON object ，即包裹在“{}”里，可以表示为 key-value 结构。
// JSON 的 key 会和 URL 上的参数合并，若一个参数同时出现在 body 和 URL 上，仅取 body 上的值。
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
	setRequestBodyDescription(state, string(body))

	for k, v := range fromBody {
		// 采用先删再加的方式，使 JSON 字段尽量维持原来的样子。
		delete(lowercaseParam, strings.ToLower(k))
		lowercaseParam[k] = v
	}
	return lowercaseParam, nil
}
