package slimapi

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_slimApiLogger_Log(t *testing.T) {
	logger := NewSlimApiLogger()

	type args struct {
		url        string
		setup      func(state *webapi.ApiState)
		wantHeader string
	}

	test := func(a args) {
		if a.url == "" {
			a.url = "/"
		}

		state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, a.url, webapitest.NewStateSetup{})

		logRecorder := webapitest.NewLogRecorder()
		state.Logger = logRecorder

		if a.setup != nil {
			a.setup(state)
		}

		logger.Log(state)

		msg := logRecorder.String()
		assert.True(t, strings.HasPrefix(msg, a.wantHeader), "should start with '%v', got '%v'", a.wantHeader, msg)
	}

	t.Run("empty", func(t *testing.T) {
		test(args{
			url: "/a/b/c",
			setup: func(state *webapi.ApiState) {
				state.UserHost = "local"
			},

			wantHeader: `level=INFO message= IP=local URL=/a/b/c`,
		})
	})

	t.Run("body", func(t *testing.T) {
		test(args{
			setup: func(state *webapi.ApiState) {
				const body = "the_body"
				state.RawRequest.Body = io.NopCloser(strings.NewReader(body))
				setRequestBodyDescription(state, body)
			},

			wantHeader: `level=INFO message= IP= URL=/ Length=8 Body=the_body`,
		})
	})

	t.Run("err", func(t *testing.T) {
		test(args{
			setup: func(state *webapi.ApiState) {
				state.Error = errors.New("this is error")
			},

			wantHeader: `level=ERROR message= IP= URL=/ ErrorType=errorString Error=this is error`,
		})
	})

	t.Run("badRequest", func(t *testing.T) {
		test(args{
			setup: func(state *webapi.ApiState) {
				state.Error = webapi.CreateBadRequestError(nil, nil, "gg")
			},

			wantHeader: `level=ERROR message= IP= URL=/ ErrorType=BadRequestError Error=gg`,
		})
	})

	t.Run("bizError", func(t *testing.T) {
		test(args{
			setup: func(state *webapi.ApiState) {
				state.Error = errx.NewBizError(10000, "mm", errors.New("inner"))
			},

			wantHeader: "level=WARN message= IP= URL=/ ErrorType=BizError Error=(10000) mm\n--- ",
		})
	})

	t.Run("apiError", func(t *testing.T) {
		test(args{
			setup: func(state *webapi.ApiState) {
				state.Error = webapi.CreateApiError(nil, nil, "critical error")
			},

			wantHeader: `level=FATAL message= IP= URL=/ ErrorType=ApiError Error=critical error`,
		})
	})

	t.Run("multipartWithFields", func(t *testing.T) {
		setup := func(state *webapi.ApiState) {
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			w.WriteField("k1", "v1")
			w.WriteField("k2", "v2")

			file0, err := w.CreateFormFile("file0", "file0")
			if err != nil {
				panic(err)
			}
			file0.Write(make([]byte, 13))

			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", `form-data; name="file1"; filename="file1"`)
			header.Set("Content-Type", "image/jpeg")
			file1, err := w.CreatePart(header)
			require.NoError(t, err)
			file1.Write(make([]byte, 23))

			header = make(textproto.MIMEHeader)
			header.Set("Content-Disposition", `form-data; name="file2"; filename="jsonFile"`)
			header.Set("Content-Type", "application/json")
			file3, err := w.CreatePart(header)
			require.NoError(t, err)
			file3.Write([]byte(`{ "V": 123 }`)) // 这些空格输出时会被去掉。

			w.Close()

			req := state.RawRequest
			req.Header[webapi.HttpHeaderContentType] = []string{w.FormDataContentType()}
			req.Body = io.NopCloser(&b)
			req.ParseMultipartForm(256)

			body := map[string]any{
				"k1": "v1",
				"k2": "v2",
			}
			for k, v := range req.MultipartForm.File {
				p, err := NewFilePart(v[0])
				require.NoError(t, err)
				body[k] = p
			}

			setRequestBodyDescription(state, body)
		}

		// map 会转换为 JSON 输出，其 key 会重新排序好，是稳定的。
		bodyJson := `{
			"file0": {
				"$FileName": "file0",
				"ContentType": "application/octet-stream",
				"Size": 13
			},
			"file1": {
				"$FileName": "file1",
				"ContentType": "image/jpeg",
				"Size": 23
			},
			"file2": {
				"$FileName": "jsonFile",
				"ContentType": "application/json",
				"Size": 12,
				"Data": {"V": 123}
			},
			"k1": "v1",
			"k2": "v2"
		}`
		bodyJson = strings.ReplaceAll(bodyJson, " ", "")
		bodyJson = strings.ReplaceAll(bodyJson, "\t", "")
		bodyJson = strings.ReplaceAll(bodyJson, "\n", "")

		test(args{
			setup:      setup,
			wantHeader: `level=INFO message= IP= URL=/ ContentType=multipart/form-data Length=262 Body=` + bodyJson,
		})
	})
}
