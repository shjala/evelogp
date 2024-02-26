package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	evelog "github.com/lf-edge/eve-api/go/logs"
)

var logs []evelog.LogEntry

func in(target string, source []string) bool {
	if len(source) == 0 {
		return true
	}
	for _, s := range source {
		if s == target {
			return true
		}
	}
	return false
}

func main() {
	srcDir := flag.String("d", "", "source directory")
	sev := flag.String("l", "", "message severity, comma separated")
	src := flag.String("s", "", "messge source, comma separated")
	showRaw := flag.Bool("r", false, "show raw message content")
	flag.Parse()

	if *srcDir == "" {
		log.Fatal("source directory is required")
	}
	severity := make([]string, 0)
	if *sev != "" {
		severity = strings.Split(*sev, ",")
		if len(severity) == 0 {
			log.Fatal("severity format is incorrect")
		}
	}
	source := make([]string, 0)
	if *src != "" {
		source = strings.Split(*src, ",")
		if len(source) == 0 {
			log.Fatal("source format is incorrect")
		}
	}

	files, err := os.ReadDir(*srcDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		f, err := os.Open(*srcDir + "/" + file.Name())
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		gz, err := gzip.NewReader(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file content: %v\n", err)
			continue
		}
		defer gz.Close()

		data, err := io.ReadAll(gz)
		if err != nil {
			log.Fatal(err)
		}

		buffer := bytes.NewBuffer(data)
		for {
			line, err := buffer.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "error breaking data: %v\n", err)
				break
			}

			var entry evelog.LogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				fmt.Fprintf(os.Stderr, "error unmarshalling line: %v\n", err)
				continue
			}

			content := make(map[string]interface{})
			if err := json.Unmarshal([]byte(entry.Content), &content); err == nil {
				l := fmt.Sprintf("%s", content["level"])
				s := fmt.Sprintf("%s", content["source"])
				if in(l, severity) {
					if in(s, source) {
						logs = append(logs, entry)
					}
				}
			} else {
				if in(entry.Severity, severity) {
					// check in content
					logs = append(logs, entry)
				}
			}
		}
	}

	// Sort logs by timestamp
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.AsTime().Before(logs[j].Timestamp.AsTime())
	})

	if *showRaw {
		for _, log := range logs {
			fmt.Printf("%v\n", log)
		}
		return
	}

	for _, log := range logs {
		parsed := make(map[string]interface{})
		content := log.Content
		source := log.Source
		if err := json.Unmarshal([]byte(log.Content), &parsed); err == nil {
			if _, ok := parsed["msg"]; ok {
				content = fmt.Sprintf("%v", parsed["msg"])
			}
		}

		fmt.Printf("[%s] [%s] [%s] %s\n",
			log.Timestamp.AsTime().Format("2006-01-02 15:04:05"),
			log.Severity,
			source,
			content)

	}

}
