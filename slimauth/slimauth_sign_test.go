package slimauth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAuthorizationHeader(t *testing.T) {
	t.Run("HasVersion", func(t *testing.T) {
		res := BuildAuthorizationHeader(Authorization{
			Key:       "kk",
			Sign:      "ss",
			Timestamp: 123,
			Version:   321,
		})
		assert.Equal(t, "SLIM-AUTH Key=kk, Sign=ss, Timestamp=123, Version=321", res)
	})

	t.Run("NoVersion", func(t *testing.T) {
		res := BuildAuthorizationHeader(Authorization{
			Key:       "kk",
			Sign:      "ss",
			Timestamp: 123,
		})
		assert.Equal(t, "SLIM-AUTH Key=kk, Sign=ss, Timestamp=123", res)
	})

	t.Run("CustomScheme", func(t *testing.T) {
		res := BuildAuthorizationHeader(Authorization{
			AuthScheme: "CUSTOM",
			Key:        "kk",
			Sign:       "ss",
			Timestamp:  123,
		})
		assert.Equal(t, "CUSTOM Key=kk, Sign=ss, Timestamp=123", res)
	})
}

func TestParseAuthorizationHeader(t *testing.T) {
	do := func(header ...string) (Authorization, error) {
		r := &http.Request{
			Header: make(http.Header),
		}

		if len(header) > 0 {
			for _, v := range header {
				r.Header.Add(HttpHeaderAuthorization, v)
			}
		}

		return ParseAuthorizationHeader(r, "")
	}

	t.Run("NoHeader", func(t *testing.T) {
		_, err := do()
		require.Error(t, err)
		require.Regexp(t, "missing", err.Error())
	})

	t.Run("TooManyHeaders", func(t *testing.T) {
		_, err := do("1", "2")
		require.Error(t, err)
		require.Regexp(t, "more than one", err.Error())
	})

	t.Run("NoScheme", func(t *testing.T) {
		_, err := do("gg")
		require.Error(t, err)
		require.Regexp(t, "scheme error", err.Error())
	})

	t.Run("WrongScheme", func(t *testing.T) {
		_, err := do("Bad Key=1")
		require.Error(t, err)
		require.Regexp(t, "scheme error", err.Error())
	})

	t.Run("BadVersion", func(t *testing.T) {
		_, err := do("SLIM-AUTH Version=abc")
		require.Error(t, err)
		require.Regexp(t, "version error", err.Error())
	})

	t.Run("BadTimestamp", func(t *testing.T) {
		_, err := do("SLIM-AUTH Timestamp=abc")
		require.Error(t, err)
		require.Regexp(t, "timestamp error", err.Error())
	})

	t.Run("OK", func(t *testing.T) {
		auth, err := do("SLIM-AUTH Key=kk, Sign=ss, Timestamp=1661843240, Version=123")
		require.NoError(t, err)

		assert.Equal(t, "kk", auth.Key)
		assert.Equal(t, "ss", auth.Sign)
		assert.Equal(t, int64(1661843240), auth.Timestamp)
		assert.Equal(t, 123, auth.Version)
	})

	t.Run("DefaultVersion", func(t *testing.T) {
		auth, err := do("SLIM-AUTH Key=kk")
		require.NoError(t, err)

		assert.Equal(t, "kk", auth.Key)
		assert.Equal(t, 1, auth.Version)
	})
}

func TestParseAuthorizationHeader_customScheme(t *testing.T) {
	r := &http.Request{
		Header: make(http.Header),
	}
	r.Header.Set(HttpHeaderAuthorization, "CUSTOM Key=kk, Sign=ss, Timestamp=1661843240")

	t.Run("OK", func(t *testing.T) {
		auth, err := ParseAuthorizationHeader(r, "CUSTOM")
		require.NoError(t, err)
		assert.Equal(t, "CUSTOM", auth.AuthScheme)
	})

	t.Run("Error", func(t *testing.T) {
		_, err := ParseAuthorizationHeader(r, "")
		require.Error(t, err)
		assert.Regexp(t, "Authorization scheme error", err.Error())
	})
}

func TestHmacSha256(t *testing.T) {
	got := HmacSha256([]byte(_secret), []byte("plain to hash"))
	assert.Equal(t, "2bb18c6fa2c6859703d508842fb1ffa06b967d460d8659477a4297d31c618402", got)
}

func Test_buildDataToSign(t *testing.T) {
	t.Run("EmptyPath", func(t *testing.T) {
		r := newRequest("",
			"",
			_requestTypeGet,
			"",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_OK, typ)
		assert.Nil(t, err)

		want := "12345\nGET\n/\n\nEND"
		assert.Equal(t, want, string(data))
	})

	t.Run("SingleSlashPath", func(t *testing.T) {
		r := newRequest("",
			"/",
			_requestTypeGet,
			"",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_OK, typ)
		assert.Nil(t, err)

		want := "12345\nGET\n/\n\nEND"
		assert.Equal(t, want, string(data))
	})

	t.Run("Get", func(t *testing.T) {
		r := newRequest("",
			"/path/sub/?bb=22&D&aa=11&cc=&D&E=5&bb=44",
			_requestTypeGet,
			"",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_OK, typ)
		assert.Nil(t, err)

		// ASCII 顺序下大写字母排在小写前面。
		// 同名参数顺序需得到保证。
		want := "12345\nGET\n/path/sub/\nDD5112244cc\nEND"
		assert.Equal(t, want, string(data))
	})

	t.Run("Form", func(t *testing.T) {
		r := newRequest("",
			"/p?x=&y=",
			_requestTypeForm,
			"bb=22&aa=11&dd&&cc=33",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_OK, typ)
		assert.Nil(t, err)

		want := "12345\nPOST\n/p\nxy\n112233dd\nEND"
		assert.Equal(t, want, string(data))
	})

	t.Run("Json", func(t *testing.T) {
		r := newRequest("",
			"/p?x=x&y=y",
			_requestTypeJson,
			`{"Data":"value"}`,
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_OK, typ)
		assert.Nil(t, err)

		want := "12345\nPOST\n/p\nxy\n{\"Data\":\"value\"}\nEND"
		assert.Equal(t, want, string(data))
	})

	t.Run("ErrorBadForm", func(t *testing.T) {
		r := newRequest("",
			"",
			_requestTypeForm,
			"",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_InvalidRequestBody, typ)
		assert.Nil(t, data)
		require.Error(t, err)
	})

	t.Run("ErrorNilJsonBody", func(t *testing.T) {
		r := newRequest("",
			"",
			_requestTypeJson,
			"",
		)
		data, typ, err := buildDataToSign(r, false, 12345)
		assert.Equal(t, SignResultType_InvalidRequestBody, typ)
		assert.Nil(t, data)
		require.Error(t, err)
		require.Regexp(t, "missing body", err.Error())
	})
}

func TestAppendSign(t *testing.T) {
	r := newRequest("", "/", _requestTypeGet, "")
	signResult := AppendSign(r, "key", _secret, "SCH", _timestamp)
	require.Equal(t, SignResultType_OK, signResult.Type)

	auth, ok := r.Header[HttpHeaderAuthorization]
	require.True(t, ok)

	want := "SCH Key=key, Sign=5ad198303bf9a3ad2d6192cdb57f8d3fdead5919089dcab04f4fb914d10ed94a, Timestamp=1661934251, Version=1"
	assert.Equal(t, want, auth[0])
}

func TestSign(t *testing.T) {
	t.Run("OK-Get", func(t *testing.T) {
		r := newRequest("", "/", _requestTypeGet, "")
		signResult := Sign(r, false, _secret, _timestamp)
		assert.Equal(t, SignResultType_OK, signResult.Type)
		assert.Equal(t, "5ad198303bf9a3ad2d6192cdb57f8d3fdead5919089dcab04f4fb914d10ed94a", signResult.Sign)
	})

	t.Run("OK-Form", func(t *testing.T) {
		r := newRequest("",
			"/path?a=1&b=2",
			_requestTypeForm,
			`x=x&y=y`,
		)
		signResult := Sign(r, false, _secret, _timestamp)
		assert.Equal(t, SignResultType_OK, signResult.Type)
		assert.Equal(t, "16e4722fbedcdc6ed1b9ac368dd6612c59cca9848d638efe353d1de7009ade29", signResult.Sign)
	})

	t.Run("OK-Json", func(t *testing.T) {
		r := newRequest("",
			"/path?a=1&b=2",
			_requestTypeJson,
			`{}`,
		)
		signResult := Sign(r, false, _secret, _timestamp)
		assert.Equal(t, SignResultType_OK, signResult.Type)
		assert.Equal(t, "a126585a55869af00ca871e5b631e6c94430f20825b9881be4c7b44b84d8bf7e", signResult.Sign)
	})

	t.Run("OK-EmptyParamValue", func(t *testing.T) {
		r := newRequest("",
			"/path?a&b&c",
			_requestTypeForm,
			"x=&y=&z=",
		)
		signResult := Sign(r, false, _secret, _timestamp)
		assert.Equal(t, SignResultType_OK, signResult.Type)
		assert.Equal(t, "73c10acdc6ce9b7cb7253eaa3f918bb44a0561f1d887cd7fd4f958ea6142160d", signResult.Sign)
	})
}
