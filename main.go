package main

import (
	"bufio"
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

func main() {
	startIndex := 1
	var reader io.ReadCloser

	if len(os.Args)-1 < startIndex {
		fmt.Println("not enough arguments")
		fmt.Println(usage)
		os.Exit(1)
	}

	stdinInfo, _ := os.Stdin.Stat()
	if stdinInfo.Size() > 0 {
		reader = os.Stdin
	}

	_, err := os.Lstat(os.Args[1])
	if err == nil {
		reader, _ = os.Open(os.Args[1])
		startIndex = 2
	}

	if len(os.Args)-1 < startIndex {
		fmt.Println("not enough arguments")
		fmt.Println(usage)
		os.Exit(1)
	}

	if reader != nil {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			match := handle(scanner.Text(), startIndex)
			if match != "" {
				fmt.Println(match)
			}
		}
	}

	if reader != nil {
		reader.Close()
	}
}

func handle(line string, argIndex int) string {
	match := ""
	arg := os.Args[argIndex]

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
	default:
		fmt.Println(usage)
		os.Exit(1)
	}

	if argIndex+1 < len(os.Args) {
		return handle(match, argIndex+1)
	}

	return match
}
