package webapi

import (
	"io"
	"iter"
)

// StreamingResponse 描述以流式方式输出 HTTP body 。
type StreamingResponse interface {
	// ContentType 返回 HTTP 头的 Content-Type 的值。
	//
	// 支持的 Content-Type 比如：
	//   - text/event-stream
	//   - application/ndjson
	ContentType() string

	// Iter 返回当前实例的非泛型版本。此方法用于将泛型类型转换为 any 。
	Iter() iter.Seq2[any, error]

	// WriteJsonBlock 将 jsonBlock 按当前实例所继承的格式，写入给定的 w 。
	//
	// jsonBlock 表示一个 JSON ，它通常是 [ApiResponse] （或其衍生结构）的 JSON 序列化结果。
	WriteJsonBlock(w io.Writer, jsonBlock []byte)
}

// EventStream 表示 HTTP 回复中以 Content-Type: text/event-stream 格式传输的数据。
//
// API 方法可将此类型作为返回值，以使用 Server-Sent Events 格式，以流的形式输出结果。
//
// 一个 SSE response 的样式形如：
//
//	data: {"Code":0,"Message":"","Data":{...}}
//
//	data: {"Code":0,"Message":"","Data":{...}}
//
//	data: {"Code":10000,"Message":"error message","Data":{...}}
//
//	...
//
// 其中，每段输出通常是 [ApiResponse] （或其衍生结构）的 JSON 序列化结果。
//
// 通常出现错误时，输出流就终止了，故 error 是数据的最后一段；但并不严格要求此行为。
type EventStream[DATA any] func(yield func(data DATA, err error) bool)

var _ StreamingResponse = (*EventStream[any])(nil)

// ContentType implements [StreamingResponse.ContentType].
func (x EventStream[DATA]) ContentType() string {
	return ContentTypeEventStream
}

// WriteJsonBlock implements [StreamingResponse.WriteJsonBlock].
func (x EventStream[DATA]) WriteJsonBlock(w io.Writer, jsonBlock []byte) {
	w.Write([]byte("data: "))
	w.Write(jsonBlock)
	w.Write([]byte{'\n', '\n'})
}

// Iter implements [StreamingResponse.Iter].
func (x EventStream[DATA]) Iter() iter.Seq2[any, error] {
	return func(yield func(data any, err error) bool) {
		for d, e := range x {
			if !yield(d, e) {
				return
			}
		}
	}
}

// NdJson 表示 HTTP 回复中以 Content-Type: application/x-ndjson 格式传输的数据。
//
// API 方法可将此类型作为返回值，以使用 Newline Delimited JSON 格式，以流的形式输出结果。
//
// 一个 NDJSON response 的样式形如：
//
//	{"Code":0,"Message":"","Data":{...}}
//	{"Code":0,"Message":"","Data":{...}}
//	{"Code":10000,"Message":"error message","Data":{...}}
//	...
//
// 其中，每段输出通常是 [ApiResponse] （或其衍生结构）的 JSON 序列化结果。
//
// 通常出现错误时，输出流就终止了，故 error 是数据的最后一段；但并不严格要求此行为。
type NdJson[DATA any] func(yield func(data DATA, err error) bool)

var _ StreamingResponse = (*NdJson[any])(nil)

// ContentType implements [StreamingResponse.ContentType].
func (x NdJson[DATA]) ContentType() string {
	return ContentTypeNdJson
}

// WriteJsonBlock implements [StreamingResponse.WriteJsonBlock].
func (x NdJson[DATA]) WriteJsonBlock(w io.Writer, jsonBlock []byte) {
	w.Write(jsonBlock)
	w.Write([]byte{'\n'})
}

// Iter implements [StreamingResponse.Iter].
func (x NdJson[DATA]) Iter() iter.Seq2[any, error] {
	return func(yield func(data any, err error) bool) {
		for d, e := range x {
			if !yield(d, e) {
				return
			}
		}
	}
}
