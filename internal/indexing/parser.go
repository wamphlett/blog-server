package indexing

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	log "unknwon.dev/clog/v2"
)

func (i *Index) parseFileHeaders(path string) (headers map[string]string) {
	headers = make(map[string]string)
	log.Info("parsing file headers: %s", path)
	file, err := os.Open(path)
	if err != nil {
		_ = bugsnag.Notify(errors.Wrap(err, "failed to parse file headers"))
		return
	}
	defer file.Close()

	startTime := time.Now()
	defer i.metrics.ParseHeaders(startTime)

	// scan the top of the file to look for a comment block containing the tags
	scanner := bufio.NewScanner(file)
	firstLine := true
	for scanner.Scan() {
		t := scanner.Text()
		if firstLine {
			if !strings.Contains(t, "<!--") {
				log.Warn("missing headers from file: %s", path)
				return
			}
			firstLine = false
			continue
		}
		if strings.Contains(t, "-->") {
			return
		}
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			log.Warn("invalid headers in file: %s (%s)", path, t)
			_ = bugsnag.Notify(errors.New(fmt.Sprintf("invalid headers in file: %s (%s)", path, t)), bugsnag.MetaData{
				"file": {
					"path": path,
				},
			})

			continue
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return
}
