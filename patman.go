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

var input string
var index string
var format string
var mem int
var help bool
var pipelines [][]Command
var pipelineNames []string
var delimiter string
var joinDelimiter string
var stdoutBufferSize int

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&index, "index", "", "index property used to aggregate logs")
	flag.StringVar(&format, "format", "stdout", "format to be used for output, pipelines are printed in order")
	flag.BoolVar(&help, "help", false, "shows help message")
	flag.BoolVar(&help, "h", false, "shows help message")
	flag.IntVar(&mem, "mem", 10, "Buffer size in MB")
	flag.StringVar(&delimiter, "delimiter", "", "split input into a sequence of lines using a custom delimiter")
	flag.StringVar(&joinDelimiter, "join", "", "join output using a custom delimiter. Writes to stdout")
	flag.IntVar(&stdoutBufferSize, "buffer", 0, "flush stdout in batches to increase performance")
}

func Run() {
	flag.Parse()

	if help {
		flag.Usage()
		usage()
		os.Exit(0)
	}

	for _, raw := range os.Args[1:] {
		var ops []string
		for key := range operators {
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

	if joinDelimiter != "" {
		print = handleJoinPrint
	}

	if stdoutBufferSize > 0 {
		print = handleBufferedStdoutPrint
		defer flushBufferedStdout()
	}

	usedMem := mem * 1024 * 1024
	buf := make([]byte, 0, usedMem)
	scanner.Buffer(buf, usedMem)

	if delimiter != "" {
		scanner.Split(ScanDelimiter(delimiter))
	} else {
		scanner.Split(bufio.ScanLines)
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
			// TODO: Double check if sorting is broken after generics
			slices.SortFunc(results, func(a, b []string) int {
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
					return 1
				}
				if bIndex < 0 {
					return -1
				}
				return aIndex - bIndex
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
		match = operators[cmd.Name].Operator(line, cmd.Arg)
	}

	if len(cmds) > 1 {
		return handle(match, cmds[1:])
	}

	return match, name
}

func usage() {
	fmt.Println("Available commands:")
	for name, entry := range operators {
		if entry.Usage == "" {
			continue
		}

		cmd := name
		if entry.Alias != "" {
			cmd = name + ", " + entry.Alias
		}
		fmt.Println("  ", cmd)

		if entry.Usage != "" {
			fmt.Println("       ", entry.Usage)
		}
		if entry.Example != "" {
			fmt.Println("        e.g.", entry.Example)
		}
	}
}
