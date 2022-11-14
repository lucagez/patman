package patman

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type transformer func(line string, arg string) string

var transformers = map[string]transformer{
	"match":        handleMatch,
	"m":            handleMatch,
	"matchall":     handleMatchAll,
	"ma":           handleMatchAll,
	"replace":      handleReplace,
	"r":            handleReplace,
	"matchline":    handleMatchLine,
	"ml":           handleMatchLine,
	"notmatchline": handleNotMatchLine,
	"nml":          handleNotMatchLine,
	"split":        handleSplit,
	"s":            handleSplit,
	// "js":           handleJs,
}

func Register(name string, t transformer) {
	transformers[name] = t
}

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
	// TODO: How to make this useful in case `/` needs to be matched?
	cmds := strings.Split(command, "/")
	pattern, replacement := cmds[0], cmds[1]
	regex, err := regexp.Compile(cmds[0])
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", pattern)
		os.Exit(1)
	}

	// attempt replace with named captures
	if strings.Contains(replacement, `%`) {
		submatches := regex.FindStringSubmatch(line)
		names := regex.SubexpNames()
		if len(submatches) == 0 {
			return ""
		}
		for i, match := range submatches {
			if i != 0 && match != "" && names[i] != "" {
				replacement = strings.ReplaceAll(replacement, "%"+names[i], match)
			}
		}
		return regex.ReplaceAllString(line, replacement)
	}

	return regex.ReplaceAllString(line, replacement)
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

func handleSplit(line, command string) string {
	cmds := strings.Split(command, "/")
	pattern, arg := cmds[0], cmds[1]
	regex, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", command)
		os.Exit(1)
	}
	index, err := strconv.ParseInt(arg, 10, 32)
	if err != nil {
		fmt.Printf("`%s` is not a valid index\n", arg)
		os.Exit(1)
	}

	parts := regex.Split(line, -1)
	if len(parts)-1 < int(index) {
		fmt.Printf("Trying to access out of range index `%d` on:\n", index)
		fmt.Println(parts)
		os.Exit(1)
	}

	return parts[index]
}

// var vm *goja.Runtime

// func handleJs(line, command string) string {
// 	if vm == nil {
// 		vm = goja.New()
// 	}

// 	if !strings.HasPrefix(command, ".") {
// 		command = "." + command
// 	}

// 	// TODO: Should probably escape
// 	script := fmt.Sprintf("String(`%s`%s)", line, command)
// 	v, err := vm.RunString(script)
// 	if err != nil {
// 		fmt.Println("error while executing js pipeline:")
// 		fmt.Println(" ", err)
// 		fmt.Println("")
// 		fmt.Println(" ", "pipeline 👉", command)
// 		fmt.Println(" ", "on line 👉", line)
// 		os.Exit(1)
// 	}
// 	return v.Export().(string)
// }
