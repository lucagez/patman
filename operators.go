package patman

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

type OperatorEntry struct {
	Operator operator
	Usage    string
	Alias    string
	Example  string
}

type operator func(line string, arg string) string

var operators = map[string]OperatorEntry{
	"match": {
		Operator: handleMatch,
		Usage:    "matches first instance that satisfies expression",
		Example:  "echo hello | match(e(.*)) # -> ello",
		Alias:    "m",
	},
	"m": {
		Operator: handleMatch,
	},
	"matchall": {
		Operator: handleMatchAll,
		Usage:    "matches all instances that satisfy expression",
		Example:  "echo hello | matchall(l) # -> ll",
		Alias:    "ma",
	},
	"ma": {
		Operator: handleMatchAll,
	},
	"replace": {
		Operator: handleReplace,
		Usage:    "replaces expression with provided string",
		Example:  "echo hello | replace(e/a) # -> hallo",
		Alias:    "r",
	},
	"r": {
		Operator: handleReplace,
	},
	"matchline": {
		Operator: handleMatchLine,
		Usage:    "matches entire line that satisfies expression",
		Example:  "cat test.txt | matchline(hello) # -> ... matching lines",
		Alias:    "ml",
	},
	"ml": {
		Operator: handleMatchLine,
	},
	"notmatchline": {
		Operator: handleNotMatchLine,
		Usage:    "returns entire lines that do not match expression",
		Example:  "cat test.txt | matchline(hello) # -> ... matching lines",
		Alias:    "nml",
	},
	"nml": {
		Operator: handleNotMatchLine,
	},
	"split": {
		Operator: handleSplit,
		Usage:    "split line by provided delimiter and take provided index",
		Example:  "echo 'a b c' | split(\\s/1) # -> b",
		Alias:    "s",
	},
	"s": {
		Operator: handleSplit,
	},
	"js": {
		Operator: handleJs,
		Usage:    "execute js expression by passing `x` as argument. returned value is coerced to string",
		Example:  "echo hello | js(x + 123) # -> hello123",
	},
}

func Register(name string, o OperatorEntry) {
	operators[name] = o
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

var vm *goja.Runtime

func handleJs(line, command string) string {
	if vm == nil {
		vm = goja.New()
	}

	// TODO: Should probably escape
	vm.RunString(fmt.Sprintf("x = `%s`", line))
	script := fmt.Sprintf("String(%s)", command)
	v, err := vm.RunString(script)
	if err != nil {
		fmt.Println("error while executing js operator:")
		fmt.Println(" ", err)
		fmt.Println("")
		fmt.Println(" ", "command:", command)
		fmt.Println(" ", "line:", line)
		os.Exit(1)
	}
	return v.Export().(string)
}
