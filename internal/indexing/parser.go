package indexing

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func (i *Index) parseFileHeaders(path string) (headers map[string]string) {
	headers = make(map[string]string)
	log.Printf("parsing file headers: %s", path)
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	firstLine := true
	for scanner.Scan() {
		t := scanner.Text()
		if firstLine {
			if !strings.Contains(t, "<!--") {
				log.Printf("missing headers from file: %s", path)
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
			log.Printf("invalid headers in file: %s (%s)", path, t)
			continue
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return
}
