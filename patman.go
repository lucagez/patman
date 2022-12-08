package patman

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"
)

// TODO: create better usage message
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
var index string
var format string
var help bool
var pipelines [][]Command
var pipelineNames []string

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&index, "index", "", "index property used to aggregate logs")
	flag.StringVar(&format, "format", "stdout", "format to be used for output, pipelines are printed in order")
	flag.BoolVar(&help, "help", false, "shows help message")
	flag.BoolVar(&help, "h", false, "shows help message")
}

func Run() {
	flag.Parse()

	if help {
		fmt.Println(usage)
		os.Exit(0)
	}

	for _, raw := range os.Args[1:] {
		var ops []string
		for key := range transformers {
			ops = append(ops, key)
		}
		if !regexp.MustCompile("(" + strings.Join(ops, "|") + ")\\(").MatchString(raw) {
			continue
		}

		parser := NewParser(raw)
		cmds, err := parser.Parse()
		if err != nil {
			log.Fatal(err)
		}

		for _, cmd := range cmds {
			if cmd.Name == "name" {
				pipelineNames = append(pipelineNames, cmd.Arg)
			}
		}

		pipelines = append(pipelines, cmds)
	}

	if index != "" {
		indexPipeline := false
		for _, name := range pipelineNames {
			if name == index {
				indexPipeline = true
			}
		}

		if !indexPipeline {
			fmt.Printf("index `%s` must have a matching named pipeline\n", index)
			os.Exit(1)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	f, openFileErr := os.Open(input)
	if openFileErr == nil {
		scanner = bufio.NewScanner(f)
	}

	var print printer
	for name, p := range printers {
		if name == format {
			print = p
		}
	}

	// using custom formats in all other cases
	if print == nil && format != "" {
		print = handleCustomFormatPrint
	}

	for scanner.Scan() {
		var results [][]string // match, name
		for _, pipeline := range pipelines {
			match, name := handle(scanner.Text(), pipeline)
			if match != "" {
				results = append(results, []string{match, name})
			}
		}

		if len(pipelineNames) > 0 {
			slices.SortFunc(results, func(a, b []string) bool {
				// Unnamed pipelines should be pushed last
				aIndex := -1
				bIndex := -1
				for i, name := range pipelineNames {
					if name == a[1] {
						aIndex = i
					}
					if name == b[1] {
						bIndex = i
					}
				}
				if aIndex < 0 {
					return false
				}
				if bIndex < 0 {
					return true
				}
				return aIndex < bIndex
			})
		}

		// do not perform in memory buffering if no index is provided
		if index == "" {
			print(results)
			continue
		}

		buffered := buffer(results)
		if buffered != nil {
			print(buffered)
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if openFileErr == nil {
		f.Close()
	}
}

func handle(line string, cmds []Command) (string, string) {
	match := ""
	name := ""
	cmd := cmds[0]

	if cmd.Name == "name" {
		name = cmd.Arg
		match = line
	} else {
		match = transformers[cmd.Name](line, cmd.Arg)
	}

	if len(cmds) > 1 {
		return handle(match, cmds[1:])
	}

	return match, name
}
