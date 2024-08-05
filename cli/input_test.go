package cli

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func WithFakeStdin(data []byte, mode fs.FileMode, f func()) {
	fs := fstest.MapFS{
		"stdin": {
			Data: data,
			Mode: mode,
		},
	}
	stdinFile, _ := fs.Open("stdin")
	Stdin = stdinFile
	defer func() { Stdin = os.Stdin }()
	f()
}

func TestInputStructuredJSON(t *testing.T) {
	WithFakeStdin([]byte{}, fs.ModeCharDevice, func() {
		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err := SetBody("application/json", []string{"foo: 1, bar: false"}, req)
		require.NoError(t, err)
		require.NotNil(t, req.Body)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"bar":false,"foo":1}`, string(body))
	})
}

func TestInputStructuredYAML(t *testing.T) {
	WithFakeStdin([]byte{}, fs.ModeCharDevice, func() {
		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err := SetBody("application/yaml", []string{"foo: 1, bar: false"}, req)
		require.NoError(t, err)
		require.NotNil(t, req.Body)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "bar: false\nfoo: 1\n", string(body))
	})
}

func TestInputBinary(t *testing.T) {
	WithFakeStdin([]byte("This is not JSON!"), 0, func() {
		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err := SetBody("", []string{}, req)
		require.NoError(t, err)
		require.NotNil(t, req.Body)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "This is not JSON!", string(body))
	})
}

func TestInputInvalidType(t *testing.T) {
	WithFakeStdin([]byte{}, fs.ModeCharDevice, func() {
		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err := SetBody("application/unknown", []string{"foo: 1"}, req)
		assert.Error(t, err)
	})
}

func TestInputFormData(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		WithFakeStdin([]byte{}, 0, func() {
			req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
			err := SetBody(
				"multipart/form-data",
				[]string{
					"key", "value",
					"filename", "@testdata/form_file.txt",
				},
				req,
			)
			require.NoError(t, err)

			assert.True(t, strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data; boundary="))

			require.NotNil(t, req.Body)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			expected := strings.Join([]string{
				`Content-Disposition: form-data; name="key"`,
				"",
				"value",
			}, "\r\n")
			assert.Contains(t, string(body), expected)
			expected = strings.Join([]string{
				`Content-Disposition: form-data; name="filename"; filename="testdata/form_file.txt"`,
				"Content-Type: application/octet-stream",
				"",
				"Hello World!",
			}, "\r\n")
			assert.Contains(t, string(body), expected)
		})
	})

	t.Run("err_args_count", func(t *testing.T) {
		WithFakeStdin([]byte{}, 0, func() {
			req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
			err := SetBody("multipart/form-data", []string{"key"}, req)
			assert.Error(t, err)
		})
	})
}
