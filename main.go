package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/elastic/go-grok"
	"github.com/elastic/go-grok/patterns"
)

// log4j PatternLayout: %d{yyyy-MM-dd HH:mm:ss,SSS} [%t] %-5p %c{1}:%L - %m%n
const grokPattern = `%{TIMESTAMP_ISO8601:timestamp} \[%{DATA:thread}\] %{LOGLEVEL:level}\s+%{JAVACLASS:logger}:%{INT:line:int} - %{GREEDYDATA:message}`

func main() {
	g := grok.New()
	if err := g.AddPatterns(patterns.Java); err != nil {
		fmt.Fprintln(os.Stderr, "add patterns:", err)
		os.Exit(1)
	}
	if err := g.Compile(grokPattern, true); err != nil {
		fmt.Fprintln(os.Stderr, "compile:", err)
		os.Exit(1)
	}

	in, err := os.Open("testdata/log4j/app.log")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer in.Close()

	out, err := os.Create("testdata/log4j/app.jsonl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	defer writer.Flush()

	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(writer)

	var current map[string]any
	flush := func() {
		if current != nil {
			if err := enc.Encode(current); err != nil {
				fmt.Fprintln(os.Stderr, "encode:", err)
				os.Exit(1)
			}
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if g.MatchString(line) {
			flush()
			captures, err := g.ParseTypedString(line)
			if err != nil {
				fmt.Fprintln(os.Stderr, "parse:", err)
				os.Exit(1)
			}
			current = captures
			continue
		}
		if current != nil {
			current["message"] = current["message"].(string) + "\n" + line
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		os.Exit(1)
	}
}
