package slimapi

import (
	"mime/multipart"
	"reflect"
	"strings"
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/webapitest"
	"github.com/stretchr/testify/require"
)

func TestFilePart_MarshalJSON(t *testing.T) {
	t.Run("binary", func(t *testing.T) {
		fh := webapitest.CreateMultipartFileHeader("name", "name", make([]byte, 3), map[string]string{
			webapi.HttpHeaderContentType: "Header1",
		})
		f, err := NewFilePart(fh)
		require.NoError(t, err)

		v, err := f.MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, `{"$FileName":"name","ContentType":"Header1","Size":3}`, string(v))
	})

	t.Run("jsonOK", func(t *testing.T) {
		body := []byte(`{"Bb":1,"Aa":2}`)
		fh := webapitest.CreateMultipartFileHeader("name", "name", body, map[string]string{
			webapi.HttpHeaderContentType: webapi.ContentTypeJson,
		})
		f, err := NewFilePart(fh)
		require.NoError(t, err)

		v, err := f.MarshalJSON()
		require.NoError(t, err)

		// 输出的 JSON key 会被重新排序。
		require.Equal(t, `{"$FileName":"name","ContentType":"application/json","Size":15,"Data":{"Aa":2,"Bb":1}}`, string(v))
	})

	t.Run("jsonBadFormat", func(t *testing.T) {
		body := []byte(`xx`)
		fh := webapitest.CreateMultipartFileHeader("name", "name", body, map[string]string{
			webapi.HttpHeaderContentType: webapi.ContentTypeJson,
		})
		_, err := NewFilePart(fh)
		require.Error(t, err)
		require.True(t, strings.HasPrefix(err.Error(), "unmarshal JSON part 'name': "), err.Error())
	})
}

func TestConv_filePartToFileHeader(t *testing.T) {
	fh := &multipart.FileHeader{Size: 100}
	in, err := NewFilePart(fh)
	require.NoError(t, err)

	res, err := Conv.ConvertType(in, reflect.TypeOf(fh))
	require.NoError(t, err)
	require.Equal(t, fh, res)
}

func TestConv_filePartToFilePart(t *testing.T) {
	fh := &multipart.FileHeader{Size: 100}
	part, err := NewFilePart(fh)
	require.NoError(t, err)

	res, err := Conv.ConvertType(part, reflect.TypeOf(part))
	require.NoError(t, err)
	require.Equal(t, part, res)
}
