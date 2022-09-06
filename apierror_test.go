package webapi

import (
	"errors"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
	"github.com/stretchr/testify/assert"
)

func TestCreateApiError(t *testing.T) {
	var e ApiError

	e = CreateApiError(nil, nil, "msg")
	assert.Equal(t, "msg", e.Error())

	e = CreateApiError(nil, nil, "msg %v %v", 1, 2)
	assert.Equal(t, "msg 1 2", e.Error())

	e = CreateApiError(nil, errors.New("inner"), "msg")
	assert.Equal(t, "msg:: inner", e.Error())

	e = CreateApiError(nil, errors.New("inner"), "")
	assert.Equal(t, "inner", e.Error())

	e = CreateApiError(nil, errors.New("inner"), "msg %v %v", 1, 2)
	assert.Equal(t, "msg 1 2:: inner", e.Error())
}

func TestDescribeError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantLevel       logx.Level
		wantName        string
		wantDescPattern []string // 有调用栈的不太好检测，用一组正则来匹配。
	}{
		{
			"nil",
			nil,
			logx.LevelInfo,
			"",
			[]string{},
		},

		{
			"normal",
			errors.New("e"),
			logx.LevelError,
			"errorString",
			[]string{"e"},
		},

		{
			"BizError-no-cause",
			errx.NewBizError(100, "msg", nil),
			logx.LevelWarn,
			"BizError",
			[]string{
				`\(100\) msg\n--- `,
				`TestDescribeError`,
			},
		},

		{
			"BizError-with-cause",
			errx.NewBizError(100, "msg", errors.New("cause")),
			logx.LevelWarn,
			"BizError",
			[]string{
				`\(100\) msg`,
				`TestDescribeError`,
				`=== cause`,
			},
		},

		{
			"ErrorWrapper",
			errx.Wrap("pre", errors.New("e")),
			logx.LevelError,
			"ErrorWrapper",
			[]string{`^pre: e\n--- `, `\n=== e\n$`},
		},

		{
			"BadRequestError",
			CreateBadRequestError(nil, errors.New("e"), "bad %v", "request"),
			logx.LevelError,
			"BadRequestError",
			[]string{`^bad request\n=== e\n$`},
		},

		{
			"ApiError",
			CreateApiError(nil, nil, "a"),
			logx.LevelFatal,
			"ApiError",
			[]string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv, name, desc := DescribeError(tt.err)
			assert.Equal(t, logx.LevelToString(tt.wantLevel), logx.LevelToString(lv))
			assert.Equal(t, tt.wantName, name)

			for _, p := range tt.wantDescPattern {
				assert.Regexp(t, p, desc)
			}
		})
	}
}
