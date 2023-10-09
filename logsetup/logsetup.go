// Package logsetup 提供一组预定义的 [webapi.LogSetup] ，以便快速实现 [webapi.ApiLogger] 。
package logsetup

import (
	"mime/multipart"
	"sort"
	"strconv"

	"github.com/cmstar/go-webapi"
)

// IP 输出发起 HTTP 请求的客户端 IP 地址。
//
// 输出字段为： IP 。
//
// 这是一个单例。
var IP = ip{}

type ip struct{}

var _ webapi.LogSetup = (*ip)(nil)

func (ip) Setup(state *webapi.ApiState) {
	state.LogMessage = append(state.LogMessage, "IP", state.UserHost)
}

// URL 输出请求的完整 URL 。
//
// 输出字段为： URL 。
//
// 这是一个单例。
var URL = url{}

type url struct{}

var _ webapi.LogSetup = (*url)(nil)

func (url) Setup(state *webapi.ApiState) {
	state.LogMessage = append(state.LogMessage, "URL", state.RawRequest.RequestURI)
}

// Error 根据当前的错误信息，判断错误的级别，并输出错误的描述信息。
//
// 输出字段为： ErrorType/Error 。
//
// 这是一个单例。
var Error = err{}

type err struct{}

var _ webapi.LogSetup = (*err)(nil)

func (err) Setup(state *webapi.ApiState) {
	if state.Error == nil {
		return
	}

	logLevel, errTypeName, errDescription := webapi.DescribeError(state.Error)

	state.LogLevel = logLevel
	state.LogMessage = append(state.LogMessage,
		"ErrorType", errTypeName,
		"Error", errDescription,
	)
}

// Files 输出 [http.Request.MultipartForm] 中的文件概要信息。
// 该字段需要先通过 [http.Request.ParseMultipartForm] 初始化。
//
// 以此输出每个文件的（X 是文件的索引）：
//   - FileX 文件名。
//   - LengthX 文件长度。
//   - ContentTypeX 文件的 Content-Type 。
//
// 不会输出非文件的部分。
//
// 这是一个单例。
var Files = files{}

type files struct{}

var _ webapi.LogSetup = (*files)(nil)

func (files) Setup(state *webapi.ApiState) {
	req := state.RawRequest
	if req.MultipartForm == nil {
		return
	}

	// 每个部分的头包括 Content-Disposition: form-data; name="photo"; filename="photo.jpeg"
	// form.File 使用的是 name ，这里按其排序，解决 map 输出顺序不稳定的问题，以便获得稳定的日志。
	sortedFileHeaders := func(form *multipart.Form) []*multipart.FileHeader {
		ln := len(form.File)
		if ln == 0 {
			return nil
		}

		keys := make([]string, 0, ln)
		for k := range form.File {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var mergedHeaders []*multipart.FileHeader
		for i := 0; i < ln; i++ {
			headers := form.File[keys[i]]
			mergedHeaders = append(mergedHeaders, headers...)
		}
		return mergedHeaders
	}

	for i, f := range sortedFileHeaders(req.MultipartForm) {
		tag := strconv.Itoa(i)
		state.LogMessage = append(state.LogMessage,
			"File"+tag, f.Filename,
			"Length"+tag, f.Size,
		)

		if f.Header != nil {
			if header, ok := f.Header[webapi.HttpHeaderContentType]; ok {
				state.LogMessage = append(state.LogMessage, "ContentType"+tag, header[0])
			}
		}
	}
}
