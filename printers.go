package patman

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/sjson"
)

type printer func(results [][]string)

var printers = map[string]printer{
	"stdout": handleStdoutPrint,
	"csv":    handleCsvPrint,
	"json":   handleJsonPrint,
}

func RegisterPrinter(name string, p printer) {
	printers[name] = p
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
	var record []string
	for _, result := range results {
		record = append(record, result[0])
		// print csv line if there's at least one non-empty value
		if result[0] != "" {
			empty = false
		}
	}

	if !empty {
		csvWriter.Write(record)
		csvWriter.Flush()
	}
}

var matchDigits = regexp.MustCompile(`^\d+(\.\d+)?$`)

func handleJsonPrint(results [][]string) {
	json := "{}"
	for _, result := range results {
		match, name := result[0], result[1]
		if name == "" {
			// TODO: This error should happen before parsing?
			fmt.Println("cannot set json without named pipeline")
			os.Exit(1)
		}

		// interpret all digit strings as numbers
		// for friendlier json serialization
		if matchDigits.MatchString(match) {
			num, _ := strconv.ParseFloat(match, 64)
			json, _ = sjson.Set(json, name, num)
			continue
		}

		if match != "" {
			json, _ = sjson.Set(json, name, match)
		}
	}
	if json != "{}" {
		fmt.Println(json)
	}
}

func handleStdoutPrint(results [][]string) {
	for i, result := range results {
		match := strings.TrimSpace(result[0])
		if match == "" {
			continue
		}
		fmt.Print(match)
		if i != len(results)-1 {
			fmt.Print(" ")
		}
	}
	if len(results) > 0 {
		fmt.Print("\n")
	}
}

func handleCustomFormatPrint(results [][]string) {
	// copy into msg
	msg := format
	for _, r := range results {
		match, name := r[0], r[1]
		// Using `%<name>` to void replacements with `$`
		msg = strings.ReplaceAll(msg, "%"+name, match)
	}
	if msg != format {
		fmt.Println(msg)
	}
}
