package reading

import (
	"bufio"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func (r *Reader) parseFileHeaders(path string) (headers map[string]string) {
	headers = make(map[string]string)
	slog.Info("parsing file headers", "path", path)
	file, err := os.Open(path)
	if err != nil {
		sentry.CaptureException(errors.Wrap(err, "failed to parse file headers"))
		return
	}
	defer file.Close()

	startTime := time.Now()
	defer r.metrics.ParseHeaders(startTime)

	// scan the top of the file to look for a comment block containing the tags
	scanner := bufio.NewScanner(file)
	firstLine := true
	inMDPropertiesBlock := false
	for scanner.Scan() {
		t := scanner.Text()
		if firstLine {
			if strings.HasPrefix(t, "---") {
				inMDPropertiesBlock = !inMDPropertiesBlock
				continue
			}

			if inMDPropertiesBlock {
				continue
			}

			if !strings.Contains(t, "<!--") {
				slog.Warn("missing headers from file", "path", path)
				return
			}
			firstLine = false
			continue
		}

		if strings.Contains(t, "-->") {
			return
		}

		// Find the first occurrence of ":"
		colonIndex := strings.Index(t, ":")

		if colonIndex == -1 {
			slog.Warn("invalid headers in file", "path", path, "line", t)
			sentry.CaptureException(errors.Errorf("invalid headers in file: %s (%s)", path, t))

			continue
		}

		// Split the line into key and value based on the first ":"
		key := strings.TrimSpace(t[:colonIndex])
		value := strings.TrimSpace(t[colonIndex+1:])

		headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return
}

func convertToTimestamp(dateStr string) int64 {
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		slog.Error("failed to parse date", "date", dateStr, "error", err)
		sentry.CaptureException(errors.Wrapf(err, "failed to parse date: %s", dateStr))
		return 0
	}
	return parsedDate.Unix()
}
