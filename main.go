package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/tidwall/sjson"
)

// RIPARTIRE QUI!<---
// - How to implement pipelines??
// - Ideally to parse data into pipelines. e.g. pattern -pipeline traceId 'm:trace_id":"(\d+)' -pipeline amount 'm:amount(\d+)' -out json -> {"traceId": 123, "amount": 345}, ...

var usage = strings.Join([]string{
	"Usage of pattern:",
	"pattern [...commands]",
	"",
	"Available commands:",
	"",
	"match, m: matches the first instance that satisfies expression",
	"  e.g. echo hello | match:e(.*) -> ello",
	"",
	"matchall, ma: matches all instances that satisfy expression",
	"  e.g. echo hello | matchall:l -> ll",
	"",
	"replace, r: replaces expression with provided string",
	"  e.g. echo hello | replace:e/a -> hallo",
	"",
	"matchline, ml: matches entire line that satisfies expression",
	"  e.g. cat test.txt | matchline:hello -> ...all matching lines",
	"",
	"notmatchline, nml: returns entire lines that do not match expression",
	"  e.g. cat test.txt | matchline:hello -> ...all matching lines",
}, "\n")

func handleMatch(line, command string) string {
	// TODO: Possible to optimize by caching regexes. PAY ATTENTION TO REGEXP SHENANIGANS
	regex, err := regexp.Compile(command)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", command)
		os.Exit(1)
	}

	return regex.FindString(line)
}

func handleMatchAll(line, command string) string {
	regex, err := regexp.Compile(command)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", command)
		os.Exit(1)
	}

	matches := ""
	for _, match := range regex.FindAllString(line, -1) {
		matches += match
	}

	return matches
}

func handleReplace(line, command string) string {
	cmds := strings.Split(command, "/")
	regex, err := regexp.Compile(cmds[0])
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", cmds[0])
		os.Exit(1)
	}

	return regex.ReplaceAllString(line, cmds[1])
}

func handleMatchLine(line, command string) string {
	regex, err := regexp.Compile(command)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", command)
		os.Exit(1)
	}
	if regex.MatchString(line) {
		return line
	}
	return ""
}

func handleNotMatchLine(line, command string) string {
	regex, err := regexp.Compile(command)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", command)
		os.Exit(1)
	}
	if !regex.MatchString(line) {
		return line
	}
	return ""
}

func handleName(line, command string) string {
	return command
}

var input string
var format string
var pipelines [][]string
var pipelineNames []string

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&format, "format", "stdout", "format to be used for output, pipelines are printed in order")
}

func main() {
	flag.Parse()

	var reader io.ReadCloser

	stdinInfo, _ := os.Stdin.Stat()
	if stdinInfo.Size() > 0 {
		reader = os.Stdin
	}

	_, err := os.Lstat(input)
	if err == nil {
		reader, _ = os.Open(input)
	}

	for _, raw := range os.Args[1:] {
		if !regexp.MustCompile(`^\w+:`).MatchString(raw) {
			continue
		}

		var cmds []string
		for _, cmd := range strings.Split(raw, "|>") {
			trimmed := strings.TrimSpace(cmd)
			cmds = append(cmds, trimmed)
			if strings.HasPrefix(trimmed, "name:") {
				pipelineNames = append(pipelineNames, strings.TrimPrefix(trimmed, "name:"))
			}
		}
		pipelines = append(pipelines, cmds)
	}

	if reader != nil {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			var results [][]string // match, name

			for _, pipeline := range pipelines {
				match, name := handle(text, pipeline)
				if match != "" {
					results = append(results, []string{match, name})
				}
			}

			if format == "stdout" {
				handleStdoutPrint(results)
			}
			if format == "json" {
				handleJsonPrint(results)
			}
			if format == "csv" {
				handleCsvPrint(results)
			}
		}
	}

	if reader != nil {
		reader.Close()
	}
}

var csvWriter *csv.Writer

func handleCsvPrint(results [][]string) {
	if len(pipelineNames) != len(pipelines) {
		// TODO: better error
		fmt.Println("all pipelines must be named")
		os.Exit(1)
	}

	if csvWriter == nil {
		csvWriter = csv.NewWriter(os.Stdout)
		csvWriter.Write(pipelineNames)
	}

	record := make([]string, len(pipelineNames))
	for _, result := range results {
		match := result[0]
		name := result[1]

		for i, pipelineName := range pipelineNames {
			if name == pipelineName {
				record[i] = match
			}
		}
	}

	csvWriter.Write(record)
	csvWriter.Flush()
}

func handleJsonPrint(results [][]string) {
	json := "{}"
	for _, result := range results {
		match := result[0]
		name := result[1]
		if name == "" {
			// TODO: This error should happen before parsing?
			fmt.Println("cannot set json without named pipeline")
			os.Exit(1)
		}
		json, _ = sjson.Set(json, name, match)
	}
	fmt.Println(json)
}

func handleStdoutPrint(results [][]string) {
	for i, result := range results {
		match := strings.TrimSpace(result[0])
		fmt.Print(match)
		if i != len(results)-1 {
			fmt.Print(" ")
		}
	}
	if len(results) > 0 {
		fmt.Print("\n")
	}
}

func handle(line string, cmds []string) (string, string) {
	match := ""
	name := ""
	arg := cmds[0]

	switch {
	case strings.HasPrefix(arg, "match:"):
		match = handleMatch(line, strings.Replace(arg, "match:", "", 1))
	case strings.HasPrefix(arg, "m:"):
		match = handleMatch(line, strings.Replace(arg, "m:", "", 1))
	case strings.HasPrefix(arg, "matchall:"):
		match = handleMatchAll(line, strings.Replace(arg, "matchall:", "", 1))
	case strings.HasPrefix(arg, "ma:"):
		match = handleMatchAll(line, strings.Replace(arg, "ma:", "", 1))
	case strings.HasPrefix(arg, "replace:"):
		match = handleReplace(line, strings.Replace(arg, "replace:", "", 1))
	case strings.HasPrefix(arg, "r:"):
		match = handleReplace(line, strings.Replace(arg, "r:", "", 1))
	case strings.HasPrefix(arg, "matchline:"):
		match = handleMatchLine(line, strings.Replace(arg, "matchline:", "", 1))
	case strings.HasPrefix(arg, "ml:"):
		match = handleMatchLine(line, strings.Replace(arg, "ml:", "", 1))
	case strings.HasPrefix(arg, "notmatchline:"):
		match = handleNotMatchLine(line, strings.Replace(arg, "notmatchline:", "", 1))
	case strings.HasPrefix(arg, "nml:"):
		match = handleNotMatchLine(line, strings.Replace(arg, "nml:", "", 1))
	case strings.HasPrefix(arg, "name:"):
		name = handleName(line, strings.Replace(arg, "name:", "", 1))
		match = line
	default:
		fmt.Println(usage)
		os.Exit(1)
	}

	if len(cmds) > 1 && match != "" {
		return handle(match, cmds[1:])
	}

	return match, name
}