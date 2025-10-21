# Patman

## Overview
Patman is a tiny command-line tool designed for processing and manipulating raw text data. It excels in tasks like log filtering and aggregation, offering a range of operators for various annoying text operations.
Its reason for existence is in all those cases where `grep` and `sed` are not enough, but a dedicated script language is overkill or too slow.

Have you ever tried to parse GBs of logs while debugging a production incident?

## Installation

### Install Script (Recommended)
Quick installation on Linux, macOS, or FreeBSD:
```bash
curl -sSfL https://raw.githubusercontent.com/lucagez/patman/main/install.sh | sh
```

Quick installation on Windows (powershell):
```bash
irm https://raw.githubusercontent.com/lucagez/patman/main/install.ps1 | iex
```

### Download Binary
Download pre-built binaries from the [releases page](https://github.com/lucagez/patman/releases/latest).

### Go Install
If you have Go installed:
```bash
go install github.com/lucagez/patman/cmd/patman@latest
```

### Build from Source
```bash
git clone https://github.com/lucagez/patman.git
cd patman
go build -o patman ./cmd/patman
sudo mv patman /usr/local/bin/
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
echo hello | patman 'match(e(.*))'  # ello
```

#### matchall/ma
Matches all instances of a regex expression.
**Usage:**
```bash
echo hello | patman 'matchall(l)'  # ll
```

#### replace/r
Replaces text matching a regex expression with a specified string.
**Usage:**
```bash
echo hello | patman 'replace(e/a)'  # hallo
```

#### named_replace/nr
Performs regex replacement using named capture groups.
**Usage:**
```bash
echo hello | patman 'named_replace(e(?P<first>l)(?P<second>l)o/%second%first)'  # ohell
```

#### matchline/ml
Matches entire lines that satisfy a regex expression.
**Usage:**
```bash
cat test.txt | patman 'matchline(hello)'  # ... matching lines
```

#### notmatchline/nml
Returns lines that do not match a regex expression.
**Usage:**
```bash
cat test.txt | patman 'notmatchline(hello)'  # ... non-matching lines
```

#### split/s
Splits a line by a specified delimiter and selects a part based on index.
**Usage:**
```bash
echo 'a b c' | patman 'split(\s/1)'  # b
```

#### filter/f
Filters lines containing a specified substring. Way faster than grep for large files.
**Usage:**
```bash
cat logs.txt | patman 'filter(hello)'  # ... matching lines
```

#### cut/c
Splits a line by delimiter and selects field(s) by index or range.
**Usage:**
```bash
echo 'a:b:c' | patman 'cut(:/0-1)'  # a:b
echo 'a:b:c' | patman 'cut(:/1)'    # b
```

#### uppercase/upper
Converts line to uppercase.
**Usage:**
```bash
echo 'hello' | patman 'uppercase()'  # HELLO
```

#### lowercase/lower
Converts line to lowercase.
**Usage:**
```bash
echo 'HELLO' | patman 'lowercase()'  # hello
```

#### uniq/u
Removes duplicate lines (keeps first occurrence).
**Usage:**
```bash
cat logs.txt | patman 'ml(error) |> uniq(_)'
```

#### gt
Filters lines that are numerically greater than the provided number.
**Usage:**
```bash
echo 101 | patman 'gt(100)' # 101
```

#### gte
Filters lines that are numerically greater than or equal to the provided number.
**Usage:**
```bash
echo 100 | patman 'gte(100)' # 100
```

#### lt
Filters lines that are numerically less than the provided number.
**Usage:**
```bash
echo 99 | patman 'lt(100)' # 99
```

#### lte
Filters lines that are numerically less than or equal to the provided number.
**Usage:**
```bash
echo 100 | patman 'lte(100)' # 100
```

#### eq
Filters lines that are numerically equal to the provided number.
**Usage:**
```bash
echo 100 | patman 'eq(100)' # 100
```

#### js
Executes a JavaScript expression, passing `x` as the argument.
**Usage:**
```bash
echo something | patman 'js(x.toUpperCase())'  # SOMETHING
echo hello | patman 'js(x + 123)'              # hello123
```

#### explode
Splits a line by a specified delimiter and joins resulting parts with a newline character.
**Usage:**
```bash
echo 'a b c' | patman 'explode(\s)'
# a
# b
# c

# Split by any character
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
