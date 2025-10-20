package patman

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

var regexCache = map[string]*regexp.Regexp{}

func regex(pattern string) *regexp.Regexp {
	if regex, ok := regexCache[pattern]; ok {
		return regex
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("`%s` is not a valid regexp pattern\n", pattern)
		os.Exit(1)
	}

	regexCache[pattern] = regex

	return regex
}

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
	"named_replace": {
		Operator: handleNamedReplace,
		Usage:    "replaces expression with provided string. Supports named capture groups",
		Example:  "echo hello | named_replace(e(?P<first>l)(?P<second>l)o / %second%first) # -> ohell",
		Alias:    "nr",
	},
	"nr": {
		Operator: handleNamedReplace,
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
	"explode": {
		Operator: handleExplode,
		Usage:    "split line by provided delimiter and join all resulting lines with a \\n (new line) char. Useful for concatenating patman with itself",
		Example:  "echo 'a b c' | explode(\\s) # -> a\nb\nc",
	},
	"filter": {
		Operator: handleFilter,
		Usage:    "matches entire line that contains substring. Useful for quickly filtering large files (> 1GB). Way quicker than cat+grep",
		Example:  "cat logs.txt | match_fast(hello) # -> ... matching lines",
		Alias:    "mf",
	},
	"f": {
		Operator: handleFilter,
	},
	"cut": {
		Operator: handleCut,
		Usage:    "split line by delimiter and select field(s) by index or range",
		Example:  "echo 'a:b:c' | cut(:/0-1) # -> a:b",
		Alias:    "c",
	},
	"c": {
		Operator: handleCut,
	},
}

func Register(name string, o OperatorEntry) {
	operators[name] = o
}

func handleMatch(line, arg string) string {
	return regex(arg).FindString(line)
}

func handleMatchAll(line, arg string) string {
	matches := ""
	for _, match := range regex(arg).FindAllString(line, -1) {
		matches += match
	}

	return matches
}

// handleNamedReplace replaces named capture groups in the replacement string.
// e.g. `echo "hello world" | named_replace(e(?P<first>l)(?P<second>l)o / %second%first) # -> ohell`
func handleNamedReplace(line, arg string) string {
	// TODO: How to make this useful in case `/` needs to be matched?
	cmds := Args(arg)
	pattern, replacement := cmds[0], cmds[1]
	re := regex(pattern)

	// attempt replace with named captures
	if strings.Contains(replacement, `%`) {
		submatches := re.FindStringSubmatch(line)
		names := re.SubexpNames()
		if len(submatches) == 0 {
			return ""
		}
		for i, match := range submatches {
			if i != 0 && match != "" && names[i] != "" {
				replacement = strings.ReplaceAll(replacement, "%"+names[i], match)
			}
		}
		return re.ReplaceAllString(line, replacement)
	}

	return re.ReplaceAllString(line, replacement)
}

func handleReplace(line, arg string) string {
	cmds := Args(arg)
	substr, replacement := cmds[0], cmds[1]

	return strings.ReplaceAll(line, substr, replacement)
}

func handleMatchLine(line, arg string) string {
	if regex(arg).MatchString(line) {
		return line
	}
	return ""
}

func handleFilter(line, arg string) string {
	if strings.Contains(line, arg) {
		return line
	}
	return ""
}

func handleNotMatchLine(line, arg string) string {
	if !regex(arg).MatchString(line) {
		return line
	}
	return ""
}

func handleSplit(line, arg string) string {
	cmds := Args(arg)
	pattern, arg := cmds[0], cmds[1]
	index, err := strconv.ParseInt(arg, 10, 32)
	if err != nil {
		fmt.Printf("`%s` is not a valid index\n", arg)
		os.Exit(1)
	}

	parts := regex(pattern).Split(line, -1)
	if len(parts)-1 < int(index) {
		return ""
	}

	return parts[index]
}

var vm *goja.Runtime

func handleJs(line, arg string) string {
	if vm == nil {
		vm = goja.New()
	}

	// TODO: Should probably escape
	vm.RunString(fmt.Sprintf("x = `%s`", line))
	script := fmt.Sprintf("String(%s)", arg)
	v, err := vm.RunString(script)
	if err != nil {
		fmt.Println("error while executing js operator:")
		fmt.Println(" ", err)
		fmt.Println("")
		fmt.Println(" ", "arg:", arg)
		fmt.Println(" ", "line:", line)
		os.Exit(1)
	}
	return v.Export().(string)
}

func handleExplode(line, arg string) string {
	cmds := Args(arg)
	pattern, arg := cmds[0], cmds[1]
	limit, err := strconv.ParseInt(arg, 10, 32)
	if err != nil {
		limit = -1
	}

	parts := regex(pattern).Split(line, int(limit))
	return strings.Join(parts, "\n")
}

func handleCut(line, arg string) string {
	cmds := Args(arg)
	delimiter := cmds[0]
	parts := regex(delimiter).Split(line, -1)
	rangeSpec := strings.Split(cmds[1], "-")
	if len(rangeSpec) == 0 {
		fmt.Printf("`%s` invalid range\n", rangeSpec)
		os.Exit(1)
	}

	var start, end int64
	var err error

	start, err = strconv.ParseInt(rangeSpec[0], 10, 32)
	if err != nil {
		fmt.Printf("`%s` is not a valid start index\n", rangeSpec[0])
		os.Exit(1)
	}

	if len(rangeSpec) == 2 {
		end, err = strconv.ParseInt(rangeSpec[1], 10, 32)
		if err != nil {
			fmt.Printf("`%s` is not a valid end index\n", rangeSpec[1])
			os.Exit(1)
		}
	}

	if start < 0 || int(start) >= len(parts) {
		return ""
	}
	if int(end) >= len(parts) {
		end = int64(len(parts) - 1)
	}
	if start > end {
		return ""
	}

	selected := parts[start : end+1]
	matches := regex(delimiter).FindAllString(line, -1)
	if len(matches) > 0 {
		return strings.Join(selected, matches[0])
	}

	return strings.Join(selected, "")
}

// Args is a utility used by operators to
// split argument by delimiter. Picking last occurrence.
// e.g. /some/url/replacement -> '/some/url' 'replacement'
func Args(arg string) []string {
	// TODO: add configuration for delimiter
	parts := strings.Split(arg, "/")
	if len(parts) < 2 {
		fmt.Printf("missing argument: %v\n", parts)
		os.Exit(1)
	}

	return []string{
		strings.Join(parts[0:len(parts)-1], "/"),
		parts[len(parts)-1],
	}
}
