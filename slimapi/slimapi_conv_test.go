package slimapi

import (
	"mime/multipart"
	"net/textproto"
	"reflect"
	"testing"

	"github.com/cmstar/go-webapi"
	"github.com/stretchr/testify/require"
)

func TestFilePart_MarshalJSON(t *testing.T) {
	fh := &multipart.FileHeader{
		Filename: "name",
		Size:     123,
		Header: textproto.MIMEHeader{
			webapi.HttpHeaderContentType: []string{"Header1", "Header2"},
		},
	}
	f, err := NewFilePart(fh)
	require.NoError(t, err)

	v, err := f.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `{"FileName":"name","ContentType":"Header1","Size":123}`, string(v))
}

func TestConv_filePartToFileHeader(t *testing.T) {
	fh := &multipart.FileHeader{Size: 100}
	in, err := NewFilePart(fh)
	require.NoError(t, err)

	res, err := Conv.ConvertType(in, reflect.TypeOf(fh))
	require.NoError(t, err)
	require.Equal(t, fh, res)
}
