package testbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestCase_WithStateBuilder(t *testing.T) {
	t.Parallel()
	// Arrange
	testcase := &TestCase[string, string, func()]{}

	// Act
	res := testcase.WithStateBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "sut"
		*state = "state"
	})

	// Assert
	assert.Equal(t, testcase, res) // pointer equal
	require.NotNil(t, testcase.StateBuilder)
	var sut string
	var state string
	testcase.StateBuilder(t, &sut, &state)
	assert.Equal(t, "sut", sut)
	assert.Equal(t, "state", state)
}

func TestTestCase_WithSpecificBuilder(t *testing.T) {
	t.Parallel()
	// Arrange
	testcase := &TestCase[string, string, func()]{}

	// Act
	res := testcase.WithSpecificBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "sut"
		*state = "state"
	})

	// Assert
	assert.Equal(t, testcase, res) // pointer equal
	require.NotNil(t, testcase.SpecificBuilder)
	var sut string
	var state string
	testcase.SpecificBuilder(t, &sut, &state)
	assert.Equal(t, "sut", sut)
	assert.Equal(t, "state", state)
}

func TestTestCase_WithAssertion(t *testing.T) {
	t.Parallel()
	// Arrange
	testcase := &TestCase[string, string, func(t *testing.T, sut string, state string)]{}

	// Act
	res := testcase.WithAssertion(func(t *testing.T, sut string, state string) {
		assert.Equal(t, "sut", sut)
		assert.Equal(t, "state", state)
	})

	// Assert
	assert.Equal(t, testcase, res) // pointer equal
	require.NotNil(t, testcase.WithAssertion)
	testcase.Assertion(t, "sut", "state")
}

func TestTestsBuilder_Register(t *testing.T) {
	t.Parallel()
	// Arrange
	builder := TestsBuilder[string, string, func(t *testing.T, sut string, state string)]{}

	// Act
	res := builder.Register("test").WithStateBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "sut"
		*state = "state"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "specific-sut"
		*state = "specific-state"
	}).WithAssertion(func(t *testing.T, sut string, state string) {
		assert.Equal(t, "specific-sut", sut)
		assert.Equal(t, "specific-state", state)
	})

	// Assert
	require.Len(t, builder.GenerateTestSets()[0].TestCases, 1)
	assert.Equal(t, builder.GenerateTestSets()[0].TestCases[0], res)
}

func TestTestsBuilder_Tests_NoTests(t *testing.T) {
	t.Parallel()
	// Arrange
	builder := TestsBuilder[string, string, func(t *testing.T, sut string, state string)]{}

	// Act
	for testName := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			// Arrange
			// nothing to do
		})
	}
}

func TestTestsBuilder_Tests_StopDuringYield(t *testing.T) {
	t.Parallel()
	// Arrange
	builder := TestsBuilder[string, string, func(t *testing.T, sut string, state string)]{}
	builder.Register("test1")
	builder.Register("test2")

	// Act
	for testName := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			// nothing to do
		})
		return
	}
}

func TestTestsBuilder_SingleTest_SpecificBuilderTakesPrecedence(t *testing.T) {
	t.Parallel()
	// Arrange
	builder := TestsBuilder[string, string, func(t *testing.T, sut string, state string)]{}
	builder.Register("test").WithStateBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "sut"
		*state = "state"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *string) {
		*sut = "specific-sut"
		*state = "specific-state"
	})

	for testName, testBuilder := range builder.Tests() {
		assert.Equal(t, "test", testName)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, "specific-sut", testData.SUT)
			assert.Equal(t, "specific-state", testData.State)
		})
	}
}

func TestTestsBuilder_MultipleTests_StateIsRepeated(t *testing.T) {
	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}
	builder.Register("1").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*state += 1
	})
	builder.Register("2").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*state += 1
	})
	builder.Register("3").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*state += 1
	})
	builder.Register("4").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*state += 1
	})

	expectedStateSequence := []int{
		1,
		2,
		3,
		4,
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedStateSequence[*indexPtr], testData.State)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedStateSequence), *indexPtr)
}

func TestTestsBuilder_MultipleTests_PreviousSpecificTestsAreIgnored(t *testing.T) {
	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}
	builder.Register("1").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut = "a"
	})
	builder.Register("2").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "b"
	})
	builder.Register("3").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "c"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut = "-" // this should not be visible in test4
	})
	builder.Register("4").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "d"
	})

	expectedRunSequence := []string{
		"a",
		"ab",
		"-",
		"abcd",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}

func TestTestsBuilder_WithSimpleAlternatives(t *testing.T) {
	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}
	builder.Register("1").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut = "a"
	})
	builder.RegisterAlternative("2").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut = "b"
	})

	expectedRunSequence := []string{
		"a",
		"b",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}

func TestTestsBuilder_WithSlightlyMoreComplexAlternatives(t *testing.T) {
	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		// Use += to capture weird edge cases
		*sut += "a"
	})
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "b"
	})
	// This registers a full new 'test-run', with all the same stuff above
	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "c"
	})

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "d"
	})

	expectedRunSequence := []string{
		"a",
		"ab",
		"abd",
		// Start of next test
		"a",
		"ac",
		"acd",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}

func TestTestsBuilder_WithComplexAlternatives(t *testing.T) {

	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "a"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "s"
	})
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "b"
	})
	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "c"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_s2"
	})
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "d"
	})
	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "e"
	})
	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "f"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_s3"
	})

	expectedRunSequence := []string{
		// Test alternative 0 0
		"as",
		"ab",
		"abd",
		"abde",
		// Test alternative 1 0
		"as",
		"ac_s2",
		"acd",
		"acde",
		// Test alternative 0 1
		"as",
		"ab",
		"abd",
		"abdf_s3",
		// Test alternative 1 1
		"as",
		"ac_s2",
		"acd",
		"acdf_s3",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}

func TestTestsBuilder_WithMultipleAlternatives(t *testing.T) {

	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "a"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "s"
	})

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "b"
	})

	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "c"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_s2"
	})

	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_alt_2"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_alt_s2"
	})

	expectedRunSequence := []string{
		// Test alternative 0
		"as",
		"ab",
		// Test alternative 1
		"as",
		"ac_s2",
		// Test alternative 2
		"as",
		"a_alt_2_alt_s2",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}
func TestTestsBuilder_WithMultipleComplexAlternatives(t *testing.T) {
	// Arrange
	builder := TestsBuilder[string, int, func(t *testing.T)]{}

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "a"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "s"
	})

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "b"
	})

	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "c"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_s2"
	})

	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_alt_2_"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "alt_s2"
	})

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "d"
	})

	builder.Register("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "e"
	})

	builder.RegisterAlternative("TEST_NAME").WithStateBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "f"
	}).WithSpecificBuilder(func(t *testing.T, sut *string, state *int) {
		*sut += "_s3"
	})

	expectedRunSequence := []string{
		// Test alternative 0 0
		"as",
		"ab",
		"abd",
		"abde",
		// Test alternative 0 1
		"as",
		"ac_s2",
		"acd",
		"acde",
		// Test alternative 0 2
		"as",
		"a_alt_2_alt_s2",
		"a_alt_2_d",
		"a_alt_2_de",
		// Test alternative 1 0
		"as",
		"ab",
		"abd",
		"abdf_s3",
		// Test alternative 1 1
		"as",
		"ac_s2",
		"acd",
		"acdf_s3",
		// Test alternative 1 2
		"as",
		"a_alt_2_alt_s2",
		"a_alt_2_d",
		"a_alt_2_df_s3",
	}

	indexNum := 0
	indexPtr := &indexNum
	for testName, testBuilder := range builder.Tests() {
		t.Run(testName, func(t *testing.T) {
			// Arrange
			testData := testBuilder(t)

			// Assert
			assert.Equal(t, expectedRunSequence[*indexPtr], testData.SUT)
			*indexPtr = *indexPtr + 1
		})
	}

	assert.Equal(t, len(expectedRunSequence), *indexPtr)
}
