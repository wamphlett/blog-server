package reading

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

type Metrics interface {
	ParseFile(startTime time.Time)
}

type Index interface {
	GetURIForFile(filepath string) string
}

type Reader struct {
	staticContentURL string
	staticContentDir string
	metrics          Metrics
	index            Index
}

func New(index Index, staticContentURL, staticContentDir string, metrics Metrics) *Reader {
	return &Reader{
		staticContentURL: staticContentURL,
		staticContentDir: staticContentDir,
		index:            index,
		metrics:          metrics,
	}
}

func (r *Reader) ReadFile(filepath string) (string, error) {
	return r.ReadFileAsHTML(filepath)
}

func (r *Reader) ReadFileAsHTML(filepath string) (string, error) {
	startTime := time.Now()
	defer r.metrics.ParseFile(startTime)

	b, err := os.ReadFile(filepath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read article file")
	}

	contents := r.replaceRelativeLinks(string(b), filepath)
	contents = r.replaceImageLinks(contents)

	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(contents), &buf); err != nil {
		_ = bugsnag.Notify(errors.Wrapf(err, "failed to parse file: %s", filepath))
	}

	return buf.String(), nil
}

func (r *Reader) replaceRelativeLinks(s, path string) string {
	reg := regexp.MustCompile(`(\[[\w\d\s\-!?]*\]\()(\.[\/\.\w\d\-]*)\)`)
	for _, match := range reg.FindAllStringSubmatch(s, -1) {
		linkedFilePath := filepath.Clean(filepath.Join(filepath.Dir(path), match[2]))
		if p := r.index.GetURIForFile(linkedFilePath); p != "" {
			s = strings.ReplaceAll(s, match[0], fmt.Sprintf("%s%s)", match[1], p))
		}
	}
	return s
}

func (r *Reader) replaceImageLinks(s string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`[\.\/]*%s\/`, r.staticContentDir))
	for _, match := range regex.FindAllString(s, -1) {
		s = strings.ReplaceAll(s, match, fmt.Sprintf("%s/%s/", r.staticContentURL, r.staticContentDir))
	}
	return s
}
