package slimapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-webapi"
)

// SlimApiInvoker 用于调用一个 SlimAPI 。
//
// TParam 是输入参数的类型； TData 对应输出的 [webapi.ApiResponse.Data] 。
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

// MustDoRaw 是 [DoRaw] 的 panic 版本。
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
func (x SlimApiInvoker[TParam, TData]) DoRaw(params TParam) (res webapi.ApiResponse[TData], err error) {
	wrapErr := func(cause error) error {
		return fmt.Errorf(`request "%s": %w`, x.Uri, cause)
	}

	in, err := json.Marshal(params)
	if err != nil {
		err = wrapErr(err)
		return
	}

	request, err := http.NewRequest(http.MethodPost, x.Uri, bytes.NewBuffer(in))
	if err != nil {
		err = wrapErr(err)
		return
	}

	request.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeJson)
	if x.RequestSetup != nil {
		err = x.RequestSetup(request)
		if err != nil {
			err = wrapErr(err)
			return
		}
	}

	response, err := new(http.Client).Do(request)
	if err != nil {
		err = wrapErr(err)
		return
	}

	defer func() {
		e := response.Body.Close()
		if err == nil {
			err = e
		}
		// Drop e if err is not nil.
	}()

	out, err := io.ReadAll(response.Body)
	if err != nil {
		err = wrapErr(err)
		return
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		err = wrapErr(err)
		return
	}

	return
}

// MustDo 是 [Do] 的 panic 版本。
func (x SlimApiInvoker[TParam, TData]) MustDo(params TParam) TData {
	res, err := x.Do(params)
	if err != nil {
		panic(err)
	}
	return res
}

// Do 执行请求并在 [webapi.ApiResponse.Code] 为 0 时返回 [webapi.ApiResponse.Data] 。
// 若 Code 不是 0 ，则返回 [errx.BizError] 。
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
