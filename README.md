# Patman

## Overview
Patman is a tiny command-line tool designed for processing and manipulating raw text data. It excels in tasks like log filtering and aggregation, offering a range of operators for various annoying text operations.
Its reason for existence is in all those cases where `grep` and `sed` are not enough, but a dedicated script language is overkill or too slow.

Have you ever tried to parse GBs of logs while debugging a production incident? 

## Installation
Currently the best way to install Patman is to use it as a library. You will need to have Go installed on your system.

```bash
go get github.com/lucagez/patman
```

Then create a `main.go` file with the following contents:
```go
package main

import "github.com/lucagez/patman"

func main() {
	patman.Run()
}
```

## Extending Patman natively

Patman can be extended with custom operators by implementing the `Operator` interface. The following example shows how to implement a uppercase operator.

```go
package main

import (
	"strings"

	"github.com/lucagez/patman"
)

func Upper(line, arg string) string {
	return strings.ToUpper(line)
}

func main() {
	patman.Register("upper", patman.OperatorEntry{
		Operator: Upper,
		Usage:    "converts line to uppercase",
		Example:  "echo 'hello' | patman 'upper(/)' # HELLO",
	})
	patman.Run()
}
```

The operator can then be used as follows:
```bash
echo hello | patman 'upper(/)' # HELLO
```

The new operator will then be available also in the `patman` command help message.

## Usage
The basic structure of a Patman command is:
```bash
patman [options] | '[operator1] |> [operator2] |> ...'
```
The `patman` command takes in a list of operators and applies them to the input data. The `|>` symbol is used to pipe the output of one operator to the next. The `patman` command can be used in a standard unix pipeline with other commands.

### Examples

Let's use as an example a log file containing the following lines:
```logs.txt
2018-01-01 00:00:00 ERROR: Something went wrong.
2018-01-01 00:00:00 INFO: Something went right.
2018-01-01 00:00:00 ERROR: Something went wrong again.
```

**Match all lines containing the word "ERROR" and replace it with "WARNING":**
```bash
cat logs.txt | patman 'matchline(ERROR) |> replace(WARNING)'
```

**Match all error lines and output a csv a timestamp and message columns:**
```bash
cat logs.txt | patman 'matchline(ERROR)' | patman -format csv \
    'split(,/1) |> name(timestamp)' \
    'split(: /1) |> name(message)'
```

This will output a csv file with the following contents:

```csv
timestamp,message
2018-01-01 00:00:00,Something went wrong.
2018-01-01 00:00:00,Something went wrong again.
```

### Initialization Options
- `-help`, `-h`: Show help message.
- `-file`: Specify the input file (default: `stdin`).
- `-index`: Define the index property for log aggregation.
- `-format`: Set the output format (default: `stdout`). One of `stdout`, `csv`, `json` or a custom formatted string.
- `-mem`: Buffer size in MB for parsing larger file chunks.
- `-delimiter`: Custom delimiter for splitting input lines.
- `-join`: Custom delimiter for joining output (default: `\n`).
- `-buffer`: Size of the stdout buffer when flushing (default: `1`).

### Operators and Aliases
Patman includes a variety of operators for text manipulation:

#### name/n
Assigns a name to the output of an operator, useful for log aggregation and naming columns in csv or json formats.
**Usage:** 
```bash
echo something | patman 'name(output_name)'
```

#### match/m
Matches the first instance of a regex expression.
**Usage:** 
```bash
echo something | patman 'match(expression)'
```

#### matchall/ma
Matches all instances of a regex expression.
**Usage:** 
```bash
echo something | patman 'matchall(expression)'
```

#### replace/r
Replaces text matching a regex expression with a specified string.
**Usage:** 
```bash
echo something | patman 'replace(expression/replacement)'
```

#### named_replace/nr
Performs regex replacement using named capture groups.
**Usage:** 
```bash
echo something | patman 'named_replace(expression/replacement)'
```

#### matchline/ml
Matches entire lines that satisfy a regex expression.
**Usage:** 
```bash
echo something | patman 'matchline(expression)'
```

#### notmatchline/nml
Returns lines that do not match a regex expression.
**Usage:** 
```bash
echo something | patman 'notmatchline(expression)'
```

#### split/s
Splits a line by a specified delimiter and selects a part based on index.
**Usage:** 
```bash
echo something | patman 'split(delimiter/index)'
```

#### filter/f
Filters lines containing a specified substring.
**Usage:** 
```bash
echo something | patman 'filter(substring)'
```

#### js
Executes a JavaScript expression, passing `x` as the argument.
**Usage:** 
```bash
echo something | patman 'js(expression)'

# e.g. uppercase
echo something | patman 'js(x.toUpperCase())' # SOMETHING
```

#### explode
Splits a line by a specified delimiter and joins resulting parts with a newline character.
**Usage:** 
```bash
echo something | patman 'explode(delimiter)'

# e.g. split by any char
echo something | patman 'explode(\.*/)'
# s
# o
# m
# e
# t
# h
# i
# n
# g
```

## License

The MIT License (MIT)
