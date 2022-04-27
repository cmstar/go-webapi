package slimapi

import (
	"strings"

	"github.com/cmstar/go-webapi"
)

// slimApiNameResolver 实现 SlimAPI 的 webapi.ApiNameResolver 。
type slimApiNameResolver struct {
}

// NewSlimApiNameResover 返回用于 SlimAPI 协议的 webapi.ApiNameResolver 实现。
func NewSlimApiNameResover() webapi.ApiNameResolver {
	return &slimApiNameResolver{}
}

// Decode 实现 webapi.ApiDecoder.Decode 。
func (d *slimApiNameResolver) FillMethod(state *webapi.ApiState) {
	/*
	 * 当前方法除填写 ApiState.Name 外，还初始化 ResponseContentType 字段，
	 * SlimAPI 的 Content-Type 是可变的，可由请求者指定，所以在解析请求时确定。
	 */

	// SlimAPI 的请求构造比较复杂，除了方法名称，还可通过 URL 指定请求格式等，需一并解析。
	// 元参数可以多种方式体现（中括号内内容为可选），优先级自上而下：
	// 形式1：http://domain/entry?~method=METHOD[&~format=FORMAT][&~callback=CALLBACK]
	// 形式2：http://domain/entry?METHOD[.FORMAT][(CALLBACK)]
	// 形式3使用路由：http://domain/entry/:~method/...[:~format]...[:~callback]
	req := state.RawRequest
	query := state.Query

	// 形式1
	method, _ := query.Get(meta_Param_Method)
	format, _ := query.Get(meta_Param_Format)
	callback, _ := query.Get(meta_Param_Callback)

	// 形式2
	if len(query.Nameless) > 0 {
		d.parseMixedMetaParams(query.Nameless, &method, &format, &callback)
	}

	// 形式3
	if method == "" {
		method = webapi.GetRouteParam(req, meta_Param_Method)
	}

	if format == "" {
		format = webapi.GetRouteParam(req, meta_Param_Format)
	}

	if callback == "" {
		callback = webapi.GetRouteParam(req, meta_Param_Callback)
	}

	// format 需要校验，如果有错整个过程就直接终止了，故首先处理。
	var requestFormat string

	// format 没有通过参数直接指定格式的情况下，尝试从 Content-Type 判断。
	if format == "" {
		contentType := req.Header.Get(webapi.HttpHeaderContentType)

		// Content-Type 可以被分号分隔，如 “text/html; charset=UTF-8”，我们只需要前面这段。
		contentType = strings.Split(contentType, ";")[0]
		contentType = strings.TrimSpace(contentType)

		switch contentType {
		case webapi.ContentTypeJson:
			requestFormat = meta_RequestFormat_Json

		case webapi.ContentTypeForm:
			requestFormat = meta_RequestFormat_Post
		}
	} else {
		// 指定的 format 串还可包含多段使用逗号隔开的值（ e.g. json,plain ）， plain 是对应回执的，需单独处理。
		parts := strings.Split(format, ",")

		for _, v := range parts {
			switch v {
			// plain 是指定返回的 Content-Type 的，所以 Content-Type 在当前方法就已经确定了，直接填上即可，不用等到 WriteResponse 。
			case meta_ResponseFormat_Plain:
				state.ResponseContentType = webapi.ContentTypePlainText

			case meta_RequestFormat_Get:
				requestFormat = meta_RequestFormat_Get

			case meta_RequestFormat_Json:
				requestFormat = meta_RequestFormat_Json

			case meta_RequestFormat_Post:
				requestFormat = meta_RequestFormat_Post

			default:
				state.Error = webapi.CreateBadRequestError(state, nil, "bad format")
				return
			}
		}
	}

	state.Name = method
	if callback != "" {
		setCallback(state, callback)
	}

	// 没指定请求格式的，默认用 GET 模式。
	if requestFormat == "" {
		requestFormat = meta_RequestFormat_Get
	}
	setRequestFormat(state, requestFormat)

	// 如果是 JSONP ，强制返回 Javascript 的 Content-Type 。
	if callback != "" {
		state.ResponseContentType = webapi.ContentTypeJavascript
	} else if state.ResponseContentType == "" {
		// 剩下的统一都是 JSON 的返回格式。
		state.ResponseContentType = webapi.ContentTypeJson
	}
}

// parseMixedMetaParams 解析 METHOD.FORMAT(CALLBACK) ，其中 .FORMAT 和 (CALLBACK) 是可选的，但顺序不能变。
// 如果没有 FORMAT 部分，则格式为： METHOD(CALLBACK) 。
//
// 解析结果填入对应的指针参数；但若该参数已经有有效值（不为 nil 且不为空字符串），则原有值不会被覆盖。
//
func (*slimApiNameResolver) parseMixedMetaParams(input string, method, format, callback *string) {
	const (
		followByFormat = iota + 1
		followByCallback
	)

	inputLen := len(input)
	pos := 0
	follow := 0

	// 确定 METHOD 后面的部分是什么。
lookup:
	for {
		c := input[pos]
		switch c {
		case '.':
			follow = followByFormat
			break lookup

		case '(':
			follow = followByCallback
			break lookup

		default:
			pos++

			// 到头了，没有 FORMAT 和 CALLBACK 部分，则整个都是 METHOD 。
			if pos == inputLen {
				break lookup
			}
		}
	}

	if method == nil || *method == "" {
		*method = input[:pos]
	}

	if follow == followByFormat {
		pos++ // Move to the next char after '.'.

		start := pos
		for {
			if input[pos] == '(' {
				follow = followByCallback
				break
			}

			pos++
			if pos == inputLen {
				break
			}
		}

		if format == nil || *format == "" {
			*format = input[start:pos]
		}
	}

	if follow == followByCallback {
		pos++ // Move to the next char after '('.

		start := pos
		for {
			if input[pos] == ')' {
				break
			}

			pos++
			if pos == inputLen {
				return
			}
		}

		if callback == nil || *callback == "" {
			*callback = input[start:pos]
		}
	}
}
