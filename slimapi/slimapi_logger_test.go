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
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func Test_slimApiLogger_Log(t *testing.T) {
	logger := NewSlimApiApiLogger()

	type stateArgs struct {
		url      string
		userHost string
		body     string
		err      error
		ctxSetup func(echo.Context)
	}

	tests := []struct {
		name       string
		args       stateArgs
		wantHeader string
	}{
		{
			name: "empty",
			args: stateArgs{
				url:      "/a/b/c",
				userHost: "local",
				body:     "",
				err:      nil,
			},
			wantHeader: `level=INFO message= Ip=local Url=/a/b/c`,
		},

		{
			name: "body",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "the_body",
				err:      nil,
			},
			wantHeader: `level=INFO message= Ip= Url=/ Length=8 Body=the_body`,
		},

		{
			name: "err",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "",
				err:      errors.New("this is error"),
			},
			wantHeader: `level=ERROR message= Ip= Url=/ ErrorType=errorString Error=this is error`,
		},

		{
			name: "badrequest",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "",
				err:      webapi.CreateBadRequestError(nil, nil, "gg"),
			},
			wantHeader: `level=ERROR message= Ip= Url=/ ErrorType=BadRequestError Error=gg`,
		},

		{
			name: "bizerror",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "",
				err:      errx.NewBizError(10000, "mm", errors.New("inner")),
			},
			wantHeader: "level=WARN message= Ip= Url=/ ErrorType=BizError Error=(10000) mm\n--- ",
		},

		{
			name: "apierror",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "",
				err:      webapi.CreateApiError(nil, nil, "critical error"),
			},
			wantHeader: `level=FATAL message= Ip= Url=/ ErrorType=ApiError Error=critical error`,
		},

		{
			name: "multipart",
			args: stateArgs{
				url:      "/",
				userHost: "",
				body:     "",
				err:      nil,
				ctxSetup: setupMultipartForm,
			},
			wantHeader: `level=INFO message= Ip= Url=/` +
				` File0=file0 Length0=13 ContentType0=application/octet-stream` +
				` File1=file1 Length1=23 ContentType1=image/jpeg`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, _ := webapitest.NewStateForTest(webapitest.NoOpHandler, tt.args.url, webapitest.NewStateSetup{})

			logRecorder := webapitest.NewLogRecorder()
			state.Logger = logRecorder
			state.UserHost = tt.args.userHost

			if tt.args.body != "" {
				setBufferedBody(state, tt.args.body)
			}

			if tt.args.err != nil {
				state.Error = tt.args.err
			}

			if tt.args.ctxSetup != nil {
				tt.args.ctxSetup(state.Ctx)
			}

			logger.Log(state)

			msg := logRecorder.String()
			assert.True(t, strings.HasPrefix(msg, tt.wantHeader), "should start with '%v', got '%v'", tt.wantHeader, msg)
		})
	}
}

func setupMultipartForm(ctx echo.Context) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("k1", "v1")
	w.WriteField("k2", "v2")

	file0, err := w.CreateFormFile("fieldName", "file0")
	if err != nil {
		panic(err)
	}
	var data0 [13]byte
	file0.Write(data0[:])

	var header textproto.MIMEHeader = make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file1"; filename="file1"`)
	header.Set("Content-Type", "image/jpeg")
	file1, err := w.CreatePart(header)
	if err != nil {
		panic(err)
	}
	var data1 [23]byte
	file1.Write(data1[:])

	w.Close()
	ctx.Request().Header.Set("Content-Type", w.FormDataContentType())
	ctx.Request().Body = io.NopCloser(&b)
	ctx.Request().ParseMultipartForm(256)
}
