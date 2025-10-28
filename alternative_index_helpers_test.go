package testbuilder

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestNewCurrIndexes(t *testing.T) {
	tests := []struct {
		name                 string
		alternativeCountList []int
		expectedOutput       string
	}{
		{
			name:                 "None iteration",
			alternativeCountList: []int{1, 1, 1},
			expectedOutput: `
000
`,
		},
		{
			name:                 "Single iteration",
			alternativeCountList: []int{1, 3, 1, 1, 1, 1},
			expectedOutput: `
000000
010000
020000
`,
		},
		{
			name:                 "Double iteration",
			alternativeCountList: []int{1, 3, 1, 2},
			expectedOutput: `
0000
0100
0200
0001
0101
0201
`,
		},
		{
			name:                 "Complex Index Iteration",
			alternativeCountList: []int{1, 3, 1, 2, 1, 2},
			expectedOutput: `
000000
010000
020000
000100
010100
020100
000001
010001
020001
000101
010101
020101
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputString strings.Builder
			currIndexes := NewCurrIndexes(tt.alternativeCountList)

			for {
				outputString.WriteString(currIndexes.String())
				outputString.WriteString("\n")
				isDone := currIndexes.AddOne()
				if isDone {
					break
				}

			}

			strippedOutput := strings.Trim(outputString.String(), "\n")
			strippedExpectedOutput := strings.Trim(tt.expectedOutput, "\n")

			assert.Equal(t, strippedExpectedOutput, strippedOutput)
		})
	}
}
