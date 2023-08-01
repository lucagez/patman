package patman

import "bytes"

func dropDelimiter(data []byte, delim []byte) []byte {
	if bytes.HasSuffix(data, delim) {
		return data[:len(data)-len(delim)]
	}
	return data
}

// ScanDelimiter is a split function for a Scanner which replaces the default split function of bufio.ScanLines.
// It can be configured via flag -sequence in order to split custom input into a sequence of lines.
// e.g. a minified js file can be split by `;` into a sequence of lines which can be processed by further by patman.
func ScanDelimiter(delimiter string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	d := []byte(delimiter)
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, d); i >= 0 {
			return i + 1, dropDelimiter(data[0:i], d), nil
		}
		if atEOF {
			return len(data), dropDelimiter(data, d), nil
		}

		// Request more data.
		return 0, nil, nil
	}
}
