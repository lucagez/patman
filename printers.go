package patman

import (
	"encoding/csv"
	"fmt"
	"os"
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

// RIPARTIRE QUI!<---
// - stdout printer should be ordered in case
//   there are indexed pipelines
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
