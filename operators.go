package patman

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"sync"

	"github.com/dop251/goja"
)

var regexCache = sync.Map{} // map[string]*regexp.Regexp

func regex(pattern string) *regexp.Regexp {
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("`%s` is not a valid regexp pattern", pattern)
	}

	regexCache.Store(pattern, re)

	return re
}

type OperatorEntry struct {
	Operator operator
	Usage    string
	Alias    string
	Example  string
}

type operator func(line string, arg string) (string, error)

var operators = map[string]OperatorEntry{
	"name": {
		Operator: handleName,
		Usage:    "assigns a name to the output of an operator, useful for log aggregation and naming columns in csv or json formats",
		Example:  "echo something | name(output_name)",
		Alias:    "n",
	},
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
		Example:  "cat logs.txt | filter(hello) # -> ... matching lines",
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
	"uppercase": {
		Operator: handleUppercase,
		Usage:    "convert line to uppercase",
		Example:  "echo 'hello' | uppercase() # -> HELLO",
		Alias:    "upper",
	},
	"upper": {
		Operator: handleUppercase,
	},
	"lowercase": {
		Operator: handleLowercase,
		Usage:    "convert line to lowercase",
		Example:  "echo 'HELLO' | lowercase() # -> hello",
		Alias:    "lower",
	},
	"lower": {
		Operator: handleLowercase,
	},
	"uniq": {
		Operator: handleUniq,
		Usage:    "remove duplicate lines (keeps first occurrence)",
		Example:  "cat logs.txt | patman 'ml(error) |> uniq(_)'",
		Alias:    "u",
	},
	"u": {
		Operator: handleUniq,
	},
	"gt": {
		Operator: handleGt,
		Usage:    "filters lines that are numerically greater than the provided number",
		Example:  "echo 101 | gt(100) # -> 101",
	},
	"gte": {
		Operator: handleGte,
		Usage:    "filters lines that are numerically greater than or equal to the provided number",
		Example:  "echo 100 | gte(100) # -> 100",
	},
	"lt": {
		Operator: handleLt,
		Usage:    "filters lines that are numerically less than the provided number",
		Example:  "echo 99 | lt(100) # -> 99",
	},
	"lte": {
		Operator: handleLte,
		Usage:    "filters lines that are numerically less than or equal to the provided number",
		Example:  "echo 100 | lte(100) # -> 100",
	},
	"eq": {
		Operator: handleEq,
		Usage:    "filters lines that are numerically equal to the provided number",
		Example:  "echo 100 | eq(100) # -> 100",
	},
}

func Register(name string, o OperatorEntry) {
	operators[name] = o
}

func handleName(line, arg string) (string, error) {
	// The name operator is a pass-through that just returns the line
	// The actual naming is handled by the command structure
	return line, nil
}

func handleMatch(line, arg string) (string, error) {
	return regex(arg).FindString(line), nil
}

func handleMatchAll(line, arg string) (string, error) {
	matches := ""
	for _, match := range regex(arg).FindAllString(line, -1) {
		matches += match
	}

	return matches, nil
}

// handleNamedReplace replaces named capture groups in the replacement string.
// e.g. `echo "hello world" | named_replace(e(?P<first>l)(?P<second>l)o / %second%first) # -> ohell`
func handleNamedReplace(line, arg string) (string, error) {
	// TODO: How to make this useful in case `/` needs to be matched?
	cmds := Args(arg)
	pattern, replacement := cmds[0], cmds[1]
	re := regex(pattern)

	// attempt replace with named captures
	if strings.Contains(replacement, `%`) {
		submatches := re.FindStringSubmatch(line)
		names := re.SubexpNames()
		if len(submatches) == 0 {
			return "", nil
		}
		for i, match := range submatches {
			if i != 0 && match != "" && names[i] != "" {
				replacement = strings.ReplaceAll(replacement, "%"+names[i], match)
			}
		}
		return re.ReplaceAllString(line, replacement), nil
	}

	return re.ReplaceAllString(line, replacement), nil
}

func handleReplace(line, arg string) (string, error) {
	cmds := Args(arg)
	pattern, replacement := cmds[0], cmds[1]

	return regex(pattern).ReplaceAllString(line, replacement), nil
}

func handleMatchLine(line, arg string) (string, error) {
	if regex(arg).MatchString(line) {
		return line, nil
	}
	return "", nil
}

func handleFilter(line, arg string) (string, error) {
	if strings.Contains(line, arg) {
		return line, nil
	}
	return "", nil
}

func handleNotMatchLine(line, arg string) (string, error) {
	if !regex(arg).MatchString(line) {
		return line, nil
	}
	return "", nil
}

func handleSplit(line, arg string) (string, error) {
	cmds := Args(arg)
	pattern, arg := cmds[0], cmds[1]
	index, err := strconv.ParseInt(arg, 10, 32)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid index", arg)
	}

	parts := regex(pattern).Split(line, -1)
	if len(parts)-1 < int(index) {
		return "", nil
	}

	return parts[index], nil
}

var vm *goja.Runtime

func handleJs(line, arg string) (string, error) {
	if workers != 1 {
		return "", fmt.Errorf("js engine does not currently supports parallelization. run again without specifying -workers flag")
	}

	if vm == nil {
		vm = goja.New()
	}

	// TODO: Should probably escape
	vm.RunString(fmt.Sprintf("x = `%s`", line))
	script := fmt.Sprintf("String(%s)", arg)
	v, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("error while executing js operator: %w (arg: %s, line: %s)", err, arg, line)
	}
	return v.Export().(string), nil
}

func handleExplode(line, arg string) (string, error) {
	cmds := Args(arg)
	pattern, arg := cmds[0], cmds[1]
	limit, err := strconv.ParseInt(arg, 10, 32)
	if err != nil {
		limit = -1
	}

	parts := regex(pattern).Split(line, int(limit))
	return strings.Join(parts, "\n"), nil
}

func handleUppercase(line, arg string) (string, error) {
	return strings.ToUpper(line), nil
}

func handleLowercase(line, arg string) (string, error) {
	return strings.ToLower(line), nil
}

var uniq sync.Map

func handleUniq(line, arg string) (string, error) {
	if _, loaded := uniq.LoadOrStore(line, true); loaded {
		return "", nil
	}
	return line, nil
}

func handleGt(line, arg string) (string, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return "", nil // Filter out non-numeric lines
	}
	limit, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid number for gt operator", arg)
	}
	if val > limit {
		return line, nil
	}
	return "", nil
}

func handleGte(line, arg string) (string, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return "", nil // Filter out non-numeric lines
	}
	limit, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid number for gte operator", arg)
	}
	if val >= limit {
		return line, nil
	}
	return "", nil
}

func handleLt(line, arg string) (string, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return "", nil // Filter out non-numeric lines
	}
	limit, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid number for lt operator", arg)
	}
	if val < limit {
		return line, nil
	}
	return "", nil
}

func handleLte(line, arg string) (string, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return "", nil // Filter out non-numeric lines
	}
	limit, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid number for lte operator", arg)
	}
	if val <= limit {
		return line, nil
	}
	return "", nil
}

func handleEq(line, arg string) (string, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return "", nil // Filter out non-numeric lines
	}
	limit, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid number for eq operator", arg)
	}
	if val == limit {
		return line, nil
	}
	return "", nil
}

// TODO: should support empty char splitting
func handleCut(line, arg string) (string, error) {
	cmds := Args(arg)
	delimiter := cmds[0]
	parts := regex(delimiter).Split(line, -1)
	rangeSpec := strings.Split(cmds[1], "-")
	if len(rangeSpec) == 0 {
		return "", fmt.Errorf("`%s` invalid range", rangeSpec)
	}

	var start, end int64
	var err error

	start, err = strconv.ParseInt(rangeSpec[0], 10, 32)
	if err != nil {
		return "", fmt.Errorf("`%s` is not a valid start index", rangeSpec[0])
	}

	if len(rangeSpec) == 2 {
		end, err = strconv.ParseInt(rangeSpec[1], 10, 32)
		if err != nil {
			return "", fmt.Errorf("`%s` is not a valid end index", rangeSpec[1])
		}
	}

	if start < 0 || int(start) >= len(parts) {
		return "", nil
	}
	if int(end) >= len(parts) {
		end = int64(len(parts) - 1)
	}
	if start > end {
		return "", nil
	}

	selected := parts[start : end+1]
	matches := regex(delimiter).FindAllString(line, -1)
	if len(matches) > 0 {
		return strings.Join(selected, matches[0]), nil
	}

	return strings.Join(selected, ""), nil
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
