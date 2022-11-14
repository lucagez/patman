package patman

// [index] => [match, name][]
var state = map[string][][]string{}

// buffer let flows all streamed records until
// they complete on matching pipelines based
// on a common index
func buffer(results [][]string) [][]string {
	var matchingIndex string

	for _, result := range results {
		match, name := result[0], result[1]

		if name == index {
			matchingIndex = match
		}
	}

	if matchingIndex == "" {
		return nil
	}

	for _, result := range results {
		if result[1] != index {
			state[matchingIndex] = append(state[matchingIndex], result)
		}
	}

	if len(state[matchingIndex]) == len(pipelineNames)-1 {
		return append([][]string{{matchingIndex, index}}, state[matchingIndex]...)
	}

	return nil
}
