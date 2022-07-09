package reading

import (
	"bytes"
	"os"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

type Reader struct {
}

func New() *Reader {
	return &Reader{}
}

func (r *Reader) ReadFile(filepath string) (string, error) {
	return r.ReadFileAsHTML(filepath)
}

func (r *Reader) ReadFileAsHTML(filepath string) (string, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read article file")
	}

	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(b, &buf); err != nil {
		panic(err)
	}

	return buf.String(), nil
}
