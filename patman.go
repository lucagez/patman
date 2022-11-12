package patman

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/tidwall/sjson"
)

// TODO: create better usage
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

var input string
var format string
var pipelines [][]string
var pipelineNames []string

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&format, "format", "stdout", "format to be used for output, pipelines are printed in order")
}

func Run() {
	flag.Parse()

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

	scanner := bufio.NewScanner(os.Stdin)

	f, openFileErr := os.Open(input)
	if openFileErr == nil {
		scanner = bufio.NewScanner(f)
	}

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

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if openFileErr == nil {
		f.Close()
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

	empty := true
	record := make([]string, len(pipelineNames))
	for _, result := range results {
		match := result[0]
		name := result[1]

		for i, pipelineName := range pipelineNames {
			if name == pipelineName {
				empty = false
				record[i] = match
			}
		}
	}

	if !empty {
		csvWriter.Write(record)
		csvWriter.Flush()
	}
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
	if json != "{}" {
		fmt.Println(json)
	}
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

	for operator, transformer := range transformers {
		prefix := fmt.Sprintf("%s:", operator)
		if strings.HasPrefix(arg, prefix) {
			match = transformer(line, strings.TrimPrefix(arg, prefix))
		}

		// to allow cases where custom operators do
		// not plan on using arguments.
		// just for a more convenient syntax
		if arg == operator {
			match = transformer(line, operator)
		}
	}

	if strings.HasPrefix(arg, "name:") {
		name = strings.Replace(arg, "name:", "", 1)
		match = line
	}

	if len(cmds) > 1 && match != "" {
		return handle(match, cmds[1:])
	}

	return match, name
}
