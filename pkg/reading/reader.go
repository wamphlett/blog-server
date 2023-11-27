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

// Metrics defines the metrics used by the reader
type Metrics interface {
	ParseFile(startTime time.Time)
	ParseHeaders(startTime time.Time)
}

// Index defines the methods required by the index
type Index interface {
	GetURIForFile(filepath string) string
}

// Reader defines a reader
type Reader struct {
	staticContentURL string
	staticContentDir string
	metrics          Metrics
	index            Index
}

// New creates a new reader with the required dependencies
func New(index Index, staticContentURL, staticContentDir string, metrics Metrics) *Reader {
	return &Reader{
		staticContentURL: staticContentURL,
		staticContentDir: staticContentDir,
		index:            index,
		metrics:          metrics,
	}
}

// ReadFileAsHTML reads the markdown file at the given location and returns the HTML version
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

// replaceRelativeLinks replaces all relative links in the content with the absolute URI
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

// replaceImageLinks replaces all the links to static files with the URL where the files are hosted
func (r *Reader) replaceImageLinks(s string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`[\.\/]*%s\/`, r.staticContentDir))
	for _, match := range regex.FindAllString(s, -1) {
		s = strings.ReplaceAll(s, match, fmt.Sprintf("%s/%s/", r.staticContentURL, r.staticContentDir))
	}
	return s
}
