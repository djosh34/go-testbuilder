package testbuilder

import (
	"strconv"
	"strings"
)


//             0 1 2 3 4 5 6 7 8 9 x l
// 1 goes into x x x x x x x x x x x x
// 3 goes into x x x x y y y y z z z z
// 1 goes into x x x x x x x x x x x x
// 2 goes into x x y y z z x x y y z z
// 1 goes into x x x x x x x x x x x x
// 2 goes into x y z x y z x y z x y z

//             0 1 2 3 4 5 6 7 8 9 x l
// 3 goes into x y z x y z x y z x y z
// 2 goes into x x x y y y x x x y y y
// 2 goes into x x x x x x y y y y y y


type IndexCounter struct {
	currIndexes []int
	alternativeCountList []int
}

func NewCurrIndexes(alternativeCountList []int) IndexCounter {
	length := len(alternativeCountList)
	currIndexes := IndexCounter{
		currIndexes: make([]int, length),
		alternativeCountList: alternativeCountList,
	}

	for i := range currIndexes.currIndexes {
		// Sanity Check
		currIndexes.currIndexes[i] = 0
	}

	return currIndexes
}

func (idx *IndexCounter) AddOne() bool {
	isDone := true

	increasedIndex := 0
	for i, currIndex := range idx.currIndexes {
		if currIndex < idx.alternativeCountList[i] - 1 {
			idx.currIndexes[i] = currIndex + 1
			increasedIndex = i
			isDone = false
			break
		}
	}

	for i := 0; i < increasedIndex; i++ {
		idx.currIndexes[i] = 0
	}


	return isDone
}

func (idx *IndexCounter) String() string {
	var outputString strings.Builder
	for _, currIndex := range idx.currIndexes {
		outputString.WriteString(strconv.Itoa(currIndex))
	}

	return outputString.String()
}