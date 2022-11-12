package patman

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

var transformers = map[string]func(string, string) string{
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
	"js":           handleJs,
}

func Register(name string, transformer func(line string, arg string) string) {
	transformers[name] = transformer
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

var vm *goja.Runtime

func handleJs(line, command string) string {
	if vm == nil {
		vm = goja.New()
	}

	if !strings.HasPrefix(command, ".") {
		command = "." + command
	}

	// TODO: Should probably escape
	script := fmt.Sprintf("String(`%s`%s)", line, command)
	v, err := vm.RunString(script)
	if err != nil {
		fmt.Println("error while executing js pipeline:")
		fmt.Println(" ", err)
		fmt.Println("")
		fmt.Println(" ", "pipeline 👉", command)
		fmt.Println(" ", "on line 👉", line)
		os.Exit(1)
	}
	return v.Export().(string)
}
