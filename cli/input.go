package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/danielgtaylor/shorthand/v2"
	"gopkg.in/yaml.v2"
)

// Stdin represents the command input, which defaults to os.Stdin.
var Stdin interface {
	Stat() (fs.FileInfo, error)
	io.Reader
} = os.Stdin

// SetBody sets the request body if one was passed either as shorthand
// arguments or via stdin.
func SetBody(mediaType string, args []string, req *http.Request) error {
	if info, err := Stdin.Stat(); err == nil {
		if len(args) == 0 && (info.Mode()&os.ModeCharDevice) == 0 {
			// There are no args but there is data on stdin. Just read it and
			// pass it through as it may not be structured data we can parse or
			// could be binary (e.g. file uploads).
			req.Body = io.NopCloser(Stdin)

			return nil
		}
	}

	if mediaType == "multipart/form-data" {
		return toFormDataBody(args, req)
	}

	input, _, err := shorthand.GetInput(args, shorthand.ParseOptions{
		EnableFileInput:       true,
		EnableObjectDetection: true,
	})
	if err != nil {
		return err
	}

	if input != nil {
		if strings.Contains(mediaType, "json") {
			marshalled, err := json.Marshal(input)
			if err != nil {
				return err
			}

			req.Body = io.NopCloser(bytes.NewReader(marshalled))
		} else if strings.Contains(mediaType, "yaml") {
			marshalled, err := yaml.Marshal(input)
			if err != nil {
				return err
			}

			req.Body = io.NopCloser(bytes.NewReader(marshalled))
		} else {
			return fmt.Errorf("not sure how to marshal %s", mediaType)
		}
	}

	return nil
}

func toFormDataBody(args []string, req *http.Request) error {
	if len(args)%2 != 0 {
		return fmt.Errorf("expected a pair number of arguments: %d", len(args))
	}

	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for i := 0; i < len(args); i += 2 {
		name := args[i]
		value := args[i+1]

		if len(value) > 1 && value[0] == '@' {
			if err := addFormFile(w, name, value[1:]); err != nil {
				return err
			}

			continue
		}

		if err := addFormField(w, name, value); err != nil {
			return err
		}
	}

	if err := w.Close(); err != nil {
		return err
	}

	req.Body = io.NopCloser(buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return nil
}

func addFormField(w *multipart.Writer, name, value string) error {
	fw, err := w.CreateFormField(name)
	if err != nil {
		return err
	}

	if _, err = fw.Write([]byte(value)); err != nil {
		return err
	}

	return nil
}

func addFormFile(w *multipart.Writer, name, filename string) error {
	fw, err := w.CreateFormFile(name, filename)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0o400)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err = io.Copy(fw, file); err != nil {
		return err
	}

	return nil
}
