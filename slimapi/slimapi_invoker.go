package slimapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-webapi"
)

// SlimApiInvoker 用于调用一个 SlimAPI 。
//
// TParam 是输入参数的类型； TData 对应输出的 [webapi.ApiResponse.Data] 。
// 若 API 返回 SSE/NDJSON 格式，则 TData 对应每一批次数据的 Data 字段。
type SlimApiInvoker[TParam, TData any] struct {
	// 目标 URL 。
	Uri string

	// 若不为 nil ，则在 [http.Client.Do] 之前，调用此函数对当前请求进行处理。
	RequestSetup func(r *http.Request) error
}

// SlimApiInvoker 创建一个 [SlimApiInvoker] 实例。
func NewSlimApiInvoker[TParam, TData any](uri string) *SlimApiInvoker[TParam, TData] {
	if uri == "" {
		panic("uri must be provided")
	}

	return &SlimApiInvoker[TParam, TData]{
		Uri: uri,
	}
}

// MustDoRaw 执行请求，并返回原始的 [webapi.ApiResponse] ，不会判断对应的 Code 值。
//
// 这是 [SlimApiInvoker.DoRaw] 的 panic 版本。
func (x SlimApiInvoker[TParam, TData]) MustDoRaw(params TParam) webapi.ApiResponse[TData] {
	res, err := x.DoRaw(params)
	if err != nil {
		panic(err)
	}
	return res
}

// DoRaw 执行请求，并返回原始的 [webapi.ApiResponse] ，不会判断对应的 Code 值。
//
// 请求总是以 Content-Type: application/json 方式发送， params 是请求的参数，需能够被 JSON 序列化。
//
// 若获得 SSE/NDJSON 流式响应，则返回错误。此时应使用 [SlimApiInvoker.DoRawStream] 等支持流式响应的方法。
func (x SlimApiInvoker[TParam, TData]) DoRaw(params TParam) (res webapi.ApiResponse[TData], err error) {
	response, err := x.request(params)
	if err != nil {
		// err 已经是包装过的，无需再包装。
		return
	}

	defer func() {
		// 对于流式输出的 API ，由于方法提前返回错误，这里未读取 body 就直接将其关闭，会影响当前连接的复用，但好过在流式内容上卡住。
		e := response.Body.Close()
		if err == nil {
			err = e
		}
		// Drop e if err is not nil.
	}()

	contentType := x.getContentType(response.Header.Get(webapi.HttpHeaderContentType))
	if contentType == webapi.ContentTypeEventStream || contentType == webapi.ContentTypeNdJson {
		err = fmt.Errorf(`request "%s": streaming response %s, use DoRawStream/MustDoStream instead`, x.Uri, contentType)
		return
	}

	out, err := io.ReadAll(response.Body)
	if err != nil {
		err = x.wrapErr(err)
		return
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		err = x.wrapErr(err)
		return
	}

	return
}

// MustDo 执行请求，并在 [webapi.ApiResponse.Code] 为 0 时返回 [webapi.ApiResponse.Data] 。
// 若 [webapi.ApiResponse.Code] 不是 0 ，则 panic [errx.BizError] 。
//
// 这是 [SlimApiInvoker.Do] 的 panic 版本。
func (x SlimApiInvoker[TParam, TData]) MustDo(params TParam) TData {
	res, err := x.Do(params)
	if err != nil {
		panic(err)
	}
	return res
}

// Do 执行请求并在 [webapi.ApiResponse.Code] 为 0 时返回 [webapi.ApiResponse.Data] 。
// 若 Code 不是 0 ，则返回 [errx.BizError] 。
//
// 若获得 SSE/NDJSON 流式响应，则返回错误。此时应使用 [SlimApiInvoker.DoRawStream] 等支持流式响应的方法。
func (x SlimApiInvoker[TParam, TData]) Do(params TParam) (data TData, err error) {
	res, err := x.DoRaw(params)
	if err != nil {
		return
	}

	if res.Code != 0 {
		cause := fmt.Errorf(`request "%s": (%d) %s`, x.Uri, res.Code, res.Message)
		err = errx.NewBizError(res.Code, res.Message, cause)
		return
	}

	data = res.Data
	return
}

// MustDoStream 是 [SlimApiInvoker.DoRawStream] 的 panic 版本，若迭代过程中出现 [webapi.ApiResponse.Code] 不是 0 ，
// 则 panic [errx.BizError] ，并结束迭代过程。
func (x SlimApiInvoker[TParam, TData]) MustDoStream(params TParam) iter.Seq[TData] {
	seq := x.DoRawStream(params)
	return func(yield func(TData) bool) {
		count := 1
		for item, err := range seq {
			if err != nil {
				panic(err)
			}

			if item.Code != 0 {
				cause := fmt.Errorf(`request "%s", seq %d: (%d) %s`, x.Uri, count, item.Code, item.Message)
				panic(errx.NewBizError(item.Code, item.Message, cause))
			}

			if !yield(item.Data) {
				return
			}

			count++
		}
	}
}

// DoRawStream 执行请求，并返回流式结果的迭代器。
//
// 规则：
//   - 若在获取第一个 [webapi.ApiResponse] 前出错（如 HTTP 请求错误），迭代器仅返回一项，错误放在该项的 error 上。
//   - 若 HTTP 响应不是流式结果，而是标准的 SlimAPI 格式，迭代器仅返回一项，包含对应的 ApiResponse ，同时 error 为 nil。
//   - 若流式响应处理过程中，出现格式错误，错误将放在迭代器结果的 error 上，迭代停止。
func (x SlimApiInvoker[TParam, TData]) DoRawStream(params TParam) iter.Seq2[webapi.ApiResponse[TData], error] {
	response, err := x.request(params)
	if err != nil {
		// err 已经是包装过的，无需再包装。
		return func(yield func(webapi.ApiResponse[TData], error) bool) {
			yield(webapi.ApiResponse[TData]{}, err)
		}
	}

	contentType := x.getContentType(response.Header.Get(webapi.HttpHeaderContentType))

	// 非流式输出，结果作为单次响应返回。
	if contentType != webapi.ContentTypeEventStream && contentType != webapi.ContentTypeNdJson {
		var err error
		var res webapi.ApiResponse[TData]
		defer func() {
			e := response.Body.Close()
			if err == nil {
				err = e
			}
			// Drop e if err is not nil.
		}()

		out, err := io.ReadAll(response.Body)
		if err != nil {
			err = x.wrapErr(err)
			return func(yield func(webapi.ApiResponse[TData], error) bool) {
				yield(webapi.ApiResponse[TData]{}, err)
			}
		}

		err = json.Unmarshal(out, &res)
		if err != nil {
			err = x.wrapErr(err)
			return func(yield func(webapi.ApiResponse[TData], error) bool) {
				yield(webapi.ApiResponse[TData]{}, err)
			}
		}

		return func(yield func(webapi.ApiResponse[TData], error) bool) {
			yield(res, nil)
		}
	}

	return func(yield func(webapi.ApiResponse[TData], error) bool) {
		body := response.Body
		defer body.Close()

		switch contentType {
		case webapi.ContentTypeEventStream:
			x.yieldFromSSE(body, yield)
		case webapi.ContentTypeNdJson:
			x.yieldFromNdJSON(body, yield)
		}
	}
}

// 执行请求，并返回状态码 200 的 Response ；否则返回错误。
func (x SlimApiInvoker[TParam, TData]) request(params TParam) (res *http.Response, errWrapped error) {
	in, err := json.Marshal(params)
	if err != nil {
		return nil, x.wrapErr(err)
	}

	request, err := http.NewRequest(http.MethodPost, x.Uri, bytes.NewBuffer(in))
	if err != nil {
		return nil, x.wrapErr(err)
	}

	request.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeJson)
	if x.RequestSetup != nil {
		err = x.RequestSetup(request)
		if err != nil {
			return nil, x.wrapErr(err)
		}
	}

	response, err := new(http.Client).Do(request)
	if err != nil {
		return nil, x.wrapErr(err)
	}

	if response.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		return nil, x.wrapErr(fmt.Errorf("unexpected HTTP status %d: %s", response.StatusCode, string(b)))
	}

	return response, nil
}

func (x SlimApiInvoker[TParam, TData]) wrapErr(cause error) error {
	return fmt.Errorf(`request "%s": %w`, x.Uri, cause)
}

func (x SlimApiInvoker[TParam, TData]) getContentType(ct string) string {
	ct = strings.TrimSpace(ct)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		return strings.TrimSpace(ct[:i])
	}
	return ct
}

func (x SlimApiInvoker[TParam, TData]) yieldFromSSE(r io.Reader, yield func(webapi.ApiResponse[TData], error) bool) {
	var eventName string
	var dataLines []string
	flush := func() bool {
		if len(dataLines) == 0 {
			eventName = ""
			return true
		}

		ev := eventName // Clone before reset.
		raw := strings.Join(dataLines, "\n")

		// Reset buffer data.
		dataLines = dataLines[:0]
		eventName = ""

		return x.dispatchStreamEnvelope(ev, raw, yield)
	}

	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			_ = yield(webapi.ApiResponse[TData]{}, err)
			return
		}

		// 移除换行符，需适配 Windows 风格的 \r\n ，从右边开始，先移除 \n ，再移除 \r 。
		line = strings.TrimRight(strings.TrimRight(line, "\n"), "\r")
		if line == "" {
			if !flush() || err == io.EOF {
				return
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			if len(dataLines) > 0 || eventName != "" {
				if !flush() {
					return
				}
			}
			return
		}
	}
}

func (x SlimApiInvoker[TParam, TData]) dispatchStreamEnvelope(eventName string, rawJSON string, yield func(webapi.ApiResponse[TData], error) bool) bool {
	var res webapi.ApiResponse[TData]
	if err := json.Unmarshal([]byte(rawJSON), &res); err != nil {
		_ = yield(webapi.ApiResponse[TData]{}, err)
		return false
	}

	if eventName == "END" || res.Code == webapi.EventStreamEndCode {
		return false
	}

	return yield(res, nil)
}

func (x SlimApiInvoker[TParam, TData]) yieldFromNdJSON(r io.Reader, yield func(webapi.ApiResponse[TData], error) bool) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		isEOF := err == io.EOF
		if err != nil && !isEOF {
			yield(webapi.ApiResponse[TData]{}, err)
			return
		}

		// 移除换行符，需适配 Windows 风格的 \r\n ，从右边开始，先移除 \n ，再移除 \r 。
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))
		if line != "" {
			var res webapi.ApiResponse[TData]
			if err = json.Unmarshal([]byte(line), &res); err != nil {
				yield(webapi.ApiResponse[TData]{}, err)
				return
			}

			if res.Code == webapi.EventStreamEndCode {
				return
			}

			if !yield(res, nil) {
				return
			}
		}

		if isEOF {
			return
		}
	}
}
