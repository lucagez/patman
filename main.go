package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
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
var index string

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&index, "index", "", "index used to aggregate matches")
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

	// PIPELINE
	var pipelines [][]string
	for _, raw := range os.Args {
		if !strings.Contains(raw, "|>") {
			continue
		}
		if !strings.Contains(raw, "name:") {
			// TODO: replace with stderr around
			fmt.Fprintf(os.Stderr, "missing `name` operand in `%s` pipeline\n", raw)
			os.Exit(1)
		}

		var cmds []string
		for _, cmd := range strings.Split(raw, "|>") {
			cmds = append(cmds, strings.TrimSpace(cmd))
		}
		pipelines = append(pipelines, cmds)
	}

	// shouldAggregate := index != ""

	// var buffer map[string][]string

	if reader != nil {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			for _, pipeline := range pipelines {
				match, name := handle(text, pipeline)
				if match != "" {
					fmt.Println(name, match)
				}
			}
		}
	}

	if reader != nil {
		reader.Close()
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

	if len(cmds) > 1 {
		return handle(match, cmds[1:])
	}

	return match, name
}
