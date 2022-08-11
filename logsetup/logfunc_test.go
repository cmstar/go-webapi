package logsetup

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/assert"
)

func TestIP(t *testing.T) {
	state := &webapi.ApiState{
		UserHost: "value",
	}
	IP.Setup(state)

	assert.Equal(t, logx.Level(0), state.LogLevel)
	assert.Len(t, state.LogMessage, 2)
	assert.Equal(t, "IP", state.LogMessage[0])
	assert.Equal(t, "value", state.LogMessage[1])
}

func TestURL(t *testing.T) {
	state := &webapi.ApiState{
		RawRequest: &http.Request{
			RequestURI: "value",
		},
	}
	URL.Setup(state)

	assert.Equal(t, logx.Level(0), state.LogLevel)
	assert.Len(t, state.LogMessage, 2)
	assert.Equal(t, "URL", state.LogMessage[0])
	assert.Equal(t, "value", state.LogMessage[1])
}

func TestError(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		state := &webapi.ApiState{}
		Error.Setup(state)

		assert.Equal(t, logx.Level(0), state.LogLevel)
		assert.Len(t, state.LogMessage, 0)
	})

	t.Run("BizError", func(t *testing.T) {
		state := &webapi.ApiState{
			Error: errx.NewBizError(100, "msg", nil),
		}
		Error.Setup(state)

		assert.Equal(t, logx.LevelWarn, state.LogLevel)
		assert.Len(t, state.LogMessage, 4)
		assert.Equal(t, "ErrorType", state.LogMessage[0])
		assert.Equal(t, "BizError", state.LogMessage[1])
		assert.Equal(t, "Error", state.LogMessage[2])
		assert.True(t, strings.Contains(state.LogMessage[3].(string), "msg"))
	})
}

func TestFiles(t *testing.T) {
	const maxMem = 10 * 1024 * 1024

	buildState := func(buf *bytes.Buffer, w *multipart.Writer) *webapi.ApiState {
		state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, "/", webapitest.NewStateSetup{
			HttpMethod:  "POST",
			ContentType: w.FormDataContentType(),
			BodyReader:  buf,
		})
		state.RawRequest.ParseMultipartForm(maxMem)
		Files.Setup(state)
		return state
	}

	t.Run("empty", func(t *testing.T) {
		state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, "/", webapitest.NewStateSetup{})
		Files.Setup(state)
		assert.Len(t, state.LogMessage, 0)
	})

	t.Run("no-file", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)
		w.WriteField("K1", "V1")
		w.Close()
		state := buildState(buf, w)

		assert.Len(t, state.LogMessage, 0)
	})

	t.Run("file", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)

		file0, _ := w.CreateFormFile("b", "f0")
		file0.Write(make([]byte, 100))

		// w.CreateFormFile() 建的文件固定是 application/octet-stream 类型的。
		// 要别的 Content-Type 得自己建。
		var header textproto.MIMEHeader = make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="a"; filename="f1"`)
		header.Set("Content-Type", "image/jpeg")
		file1, _ := w.CreatePart(header)
		file1.Write(make([]byte, 200))

		w.Close()
		state := buildState(buf, w)

		// 文件排序是按照 name: a, b ，而输出日志用的是 filename: f1, f0 。
		want := []any{
			"File0", "f1",
			"Length0", int64(200),
			"ContentType0", "image/jpeg",
			"File1", "f0",
			"Length1", int64(100),
			"ContentType1", "application/octet-stream",
		}
		assert.Equal(t, want, state.LogMessage)
	})
}
