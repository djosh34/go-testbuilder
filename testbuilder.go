// Package testbuilder provides a generic, composable mechanism for constructing
// table-driven and incremental tests in Go. It is designed for testing
// systems composed of a system under test (SUT) and an associated mutable
// state, with clear separation between shared setup logic and
// test-specific initialization.
//
// # Overview
//
// Traditional table-driven tests require manually wiring together test data,
// state setup, and assertions. In more complex systems, test scenarios often
// share setup logic and diverge only in fine-grained details. The `testbuilder`
// package automates that pattern by introducing a declarative builder that can:
//
//   - Incrementally accumulate setup logic across a sequence of registered test
//     cases (`StateBuilder`)
//   - Add per-test customization logic (`SpecificBuilder`)
//   - Produce a consistent flow of test cases through an iterator interface
//
// Each test produced by the builder receives:
//   - A freshly initialized SUT and state
//   - All cumulative setup logic up to that test
//   - A single-specific configuration for the given case
//
// This enables well-structured incremental unit tests, where each test case
// can build upon prior setup definitions, while remaining isolated at runtime.
//
// # Terminology
//
//   - SUT - “system under test,” typically a struct or object under test.
//   - STATE - any struct or variable that tracks test inputs, side effects,
//     or intermediate values used to verify behavior.
//   - ASSERT - any assertion logic or type. Users may define it as a
//     function type with a signature like `func(t *testing.T, sut SUT, state STATE, ... )`.
//
// # Example (Simple Incremental Tests)
//
// See the example in 'example_test.go' under `TestUserController_Handle` for an idiomatic usage.
//
// # Alternatives
//
// `TestsBuilder` also supports _alternative branches_, which let you multiply
// test branches for combinatorial exploration of multiple conditions.
// Alternatives are declared via `RegisterAlternative()`. When alternatives are
// present, the builder automatically generates all cross-products of test
// alternatives across registered sets.
//
// For example, if you have 3 test groups, one of which has 2 alternatives and
// another has 3, then `GenerateTestSets` produces `2 × 3 = 6` fully independent
// combinations of tests. Each generated set executes the same cumulative logic,
// but substitutes the chosen alternative within its branch.
//
package testbuilder

import (
	"fmt"
	"iter"
	"testing"
)

// TestsBuilder manages a collection of test case sets and generates all
// executable test combinations for a given SUT, STATE, and ASSERT type.
//
// Parameter Types:
//   - SUT:      the system under test (e.g., a controller or service instance)
//   - STATE:    the mutable test state that evolves across test cases
//   - ASSERT:   assertion function or logic associated with the test
//
// A builder accumulates a list of `TestCaseSet` instances, where each set
// represents a linear sequence of tests. Each set may contain one or more
// _alternatives_ (added via RegisterAlternative).
//
// A typical workflow:
//
//	builder := TestsBuilder[MySUT, MyState, MyAssertFunc]{}
//	builder.Register("case1").WithStateBuilder(...).WithAssertion(...)
//	builder.Register("case2").WithSpecificBuilder(...)
//	for name, build := range builder.Tests() {
//	    t.Run(name, func(t *testing.T) {
//	        data := build(t)
//	        // Act
//	        // data.SUT / data.State
//	        // Assert
//	        data.Assert(t, data.SUT, data.State, ...)
//	    })
//	}
//
// Conceptually, multiple test case sets result in a multi-dimensional grid of
// test combinations. Each combination of alternatives yields an independent
// branch of tests.
type TestsBuilder[SUT any, STATE any, ASSERT any] struct {
	TestCaseSets []*TestCaseSet[SUT, STATE, ASSERT]
}

// TestCaseSet groups together one or more alternative test cases. A single
// test case set represents a logical test stage (e.g., "Authentication setup")
// which can have different permutations.
//
// Each entry in `TestAlternatives` represents one possible variation or
// “alternative path” through that logical test step.
//
// Normally, you do not construct TestCaseSet manually; it is populated by
// calling `TestsBuilder.Register` and `TestsBuilder.RegisterAlternative`.
type TestCaseSet[SUT any, STATE, ASSERT any] struct {
	TestAlternatives []*TestCase[SUT, STATE, ASSERT]
}

// TestSet represents one fully concrete combination of alternatives across all
// TestCaseSets in a builder.
//
// In a scenario with multiple sets and registered alternatives, the builder
// expands all cross-products into TestSets. Each TestSet therefore represents
// one independently runnable flow of tests.
//
// For instance, if three sets exist with alternative counts [1,2,2], this
// produces 1×2×2 = 4 independent TestSets.
//
// During iteration, individual TestSets provide an ordered list of TestCases.
// Each test executes by successively applying all StateBuilders up to the
// current index, followed by the case’s SpecificBuilder.
//
// Example:
//
// - TestCase[0]: StateBuilder0 + SpecificBuilder0
// - TestCase[1]: StateBuilder0..1 + SpecificBuilder1
// - TestCase[2]: StateBuilder0..2 + SpecificBuilder2
//
// where StateBuilders accumulate cumulatively, and SpecificBuilder applies
// only once for that test.
type TestSet[SUT any, STATE any, ASSERT any] struct {
	TestCases   []*TestCase[SUT, STATE, ASSERT]
	TestSetName string
}

// TestData defines the concrete values produced for a single test run.
//
// It contains:
//   - SUT:    The instantiated system under test after all builders are applied
//   - State:  The resulting state built by the series of StateBuilders and SpecificBuilder
//   - Assert: The test’s associated assertion logic (user-defined, generic type)
//
// Although the Assert member is generic, in common usage it is a function of
// form:
//
//	func(t *testing.T, sut SUT, state STATE, results ...)
//
// allowing callers to directly invoke the expected validations after executing
// the SUT logic.
type TestData[SUT any, STATE any, ASSERT any] struct {
	// SUT represents the system under test within a test case.
	SUT SUT

	// State tracks the state produced by the TestCase.StateBuilder's
	State STATE

	// Assert function that can be specified to be any type. Typically, it is a good idea to use a function signature
	// like func(t *testing.T, state STATE, ...) where the ... is replaced by the output of the SUT
	Assert ASSERT
}

// TestCase represents one concrete test registration entry.
//
// Each TestCase defines three functional hooks:
//   - StateBuilder:    A setup method that mutates *SUT and *STATE and is applied cumulatively
//                      across all test cases registered before and including this one.
//
//   - SpecificBuilder: A one-off adjustment applied only for this specific test,
//                      always executed after all StateBuilders.
//
//   - Assertion:       Arbitrary assertion logic or function.
//
// Builders are applied in this order for each test iteration:
//
//	for i := range testCases {
//	    // Apply all prior (and current) StateBuilders
//	    for j := 0; j <= i; j++ {
//	        StateBuilder[j](sut, state)
//	    }
//	    // Apply current test’s SpecificBuilder only once
//	    SpecificBuilder[i](sut, state)
//	}
//
// This incremental design allows convenient "progressive" test authoring where
// each registered test builds upon the setup of the previous ones.
//
// Register a test via:
//
//	builder.Register("case name").WithStateBuilder(...).WithSpecificBuilder(...).WithAssertion(...)
type TestCase[SUT any, STATE any, ASSERT any] struct {
	// TestName for the test case
	TestName string
	// StateBuilder that is subsequently used to build up state for the tests. The distinction between the StateBuilder
	// and the SpecificBuilder is that StateBuilder is subsequently called for all TestCase's that are registered to the
	// TestsBuilder.
	StateBuilder func(t *testing.T, sut *SUT, state *STATE)
	// SpecificBuilder is only run for this case
	SpecificBuilder func(t *testing.T, sut *SUT, state *STATE)
	// Assertion logic
	Assertion ASSERT
}

// WithStateBuilder assigns a function that mutates SUT and STATE. The associated
// StateBuilder will run cumulatively for this and all subsequent test cases.
//
// Example:
//
//	builder.Register("common setup").
//	    WithStateBuilder(func(t *testing.T, sut *SUT, state *State) {
//	        sut.Config = "base"
//	    })
func (ts *TestCase[SUT, STATE, ASSERT]) WithStateBuilder(f func(t *testing.T, sut *SUT, state *STATE)) *TestCase[SUT, STATE, ASSERT] {
	ts.StateBuilder = f
	return ts
}

// WithSpecificBuilder defines logic that is applied *only* to this particular
// test case and does not persist to others.
//
// This is typically used to introduce isolated deviations or test-specific
// mocking behavior.
//
// Example:
//
//	builder.Register("failing case").
//	    WithSpecificBuilder(func(t *testing.T, sut *SUT, state *State) {
//	        sut.Service.FailMode = true
//	    })
func (ts *TestCase[SUT, STATE, ASSERT]) WithSpecificBuilder(f func(t *testing.T, sut *SUT, state *STATE)) *TestCase[SUT, STATE, ASSERT] {
	ts.SpecificBuilder = f
	return ts
}

// WithAssertion attaches the assertion logic to a test case. The ASSERT type is
// generic, allowing any form of validation: function callbacks, data structs,
// or test harness references.
//
// Commonly, ASSERT is defined as a function with signature:
//	func(t *testing.T, sut SUT, state STATE, results ...)
//
// Example:
//
//	builder.Register("positive").
//  WithAssertion(func(t *testing.T, sut MySUT, state MyState, result Result) {
//	    require.Nil(t, result.Err)
//	})
func (ts *TestCase[SUT, STATE, ASSERT]) WithAssertion(f ASSERT) *TestCase[SUT, STATE, ASSERT] {
	ts.Assertion = f
	return ts
}

// Register adds a new primary test case to the builder.
//
// Each Register call creates a new TestCaseSet, meaning the test becomes part
// of a new “column” in the combinatorial expansion. All previously registered
// cases remain unchanged and are included in earlier positions of generated
// sequences.
//
// Example:
//
//	builder.Register("initial state").
//	    WithStateBuilder(...)
//	builder.Register("second test").
//	    WithStateBuilder(...).WithSpecificBuilder(...)
//
// The resulting tests will incrementally call the cumulative state builders:
// - Case 1 applies only StateBuilder(1)
// - Case 2 applies StateBuilder(1) + StateBuilder(2)
func (ts *TestsBuilder[SUT, STATE, ASSERT]) Register(name string) *TestCase[SUT, STATE, ASSERT] {
	testcase := &TestCase[SUT, STATE, ASSERT]{
		TestName: name,
	}

	newTestCaseSet := &TestCaseSet[SUT, STATE, ASSERT]{
		TestAlternatives: []*TestCase[SUT, STATE, ASSERT]{
			testcase,
		},
	}

	ts.TestCaseSets = append(ts.TestCaseSets, newTestCaseSet)
	return testcase
}

// RegisterAlternative adds an *alternative* to the most recently registered
// test case set.
//
// Alternatives represent parallel variations of the most recently registered
// set and cause combinatorial expansion. Each branch of alternatives yields a
// distinct test workflow during generation.
//
// For example:
//
//	builder.Register("stage1").WithStateBuilder(...)
//	builder.RegisterAlternative("variantA").WithStateBuilder(...)
//	builder.Register("stage2")
//
// yields two independent sequences of test cases:
//   - stage1 → stage2
//   - variantA → stage2
//
// RegisterAlternative panics if called before any `Register` call.
func (ts *TestsBuilder[SUT, STATE, ASSERT]) RegisterAlternative(name string) *TestCase[SUT, STATE, ASSERT] {
	testcase := &TestCase[SUT, STATE, ASSERT]{
		TestName: name,
	}

	if len(ts.TestCaseSets) == 0 {
		// Rather have error, but then we lose backwards compatibility
		panic(fmt.Sprintf("Cannot create alternative '%s', "+
			"since no test is registered yet. "+
			"Please Use builder.Register(name) first, before defining alternatives", name))
	}

	// Get latest TestCaseSet
	latestTestCaseSet := ts.TestCaseSets[len(ts.TestCaseSets)-1]
	latestTestCaseSet.TestAlternatives = append(latestTestCaseSet.TestAlternatives, testcase)
	return testcase
}

// GenerateTestSets enumerates all possible combinations of test alternatives,
// producing one TestSet per unique combination.
//
// It constructs a cross-product across `TestCaseSets`, where each dimension’s
// size equals the number of alternatives registered in that set.
//
// Each TestSet includes one TestCase chosen from each TestCaseSet, forming one
// full path through the test graph. TestSetName is populated with an index
// representation (e.g., "0_1_2") if multiple alternatives exist.
//
// This function is primarily used internally by Tests(), but can also be
// invoked manually to inspect generated structures.
func (ts *TestsBuilder[SUT, STATE, ASSERT]) GenerateTestSets() []*TestSet[SUT, STATE, ASSERT] {
	alternativeCountList := make([]int, 0)
	moreThanOneAlternative := false

	for _, testCaseSet := range ts.TestCaseSets {
		numAlternatives := len(testCaseSet.TestAlternatives)
		if numAlternatives > 1 {
			moreThanOneAlternative = true
		}
		alternativeCountList = append(alternativeCountList, numAlternatives)
	}

	testSets := make([]*TestSet[SUT, STATE, ASSERT], 0)

	indexCounter := NewCurrIndexes(alternativeCountList)
	isDone := false
	for !isDone {
		indexes := indexCounter.currIndexes
		_ = indexes

		newTestSet := &TestSet[SUT, STATE, ASSERT]{}

		for setIdx, testcaseSet := range ts.TestCaseSets {
			altIdx := indexes[setIdx]
			testCase := testcaseSet.TestAlternatives[altIdx]
			newTestSet.TestCases = append(newTestSet.TestCases, testCase)

		}
		if moreThanOneAlternative {
			newTestSet.TestSetName = indexCounter.String()
		}
		testSets = append(testSets, newTestSet)
		isDone = indexCounter.AddOne()
	}

	return testSets
}

// Tests returns an iterator that yields all fully prepared executable test
// functions along with their corresponding test names.
//
// The iterator exposes one entry per TestCase per generated TestSet. Internally,
// the yield function constructs dynamically-built state for each test by:
//
//   1. Initializing a fresh SUT and STATE
//   2. Sequentially running all StateBuilders from TestCase[0..i]
//   3. Executing the SpecificBuilder of TestCase[i] exactly once
//
// This produces isolated, incrementally-constructed test data for every test.
//
// Example usage:
//
//	for testName, build := range builder.Tests() {
//	    t.Run(testName, func(t *testing.T) {
//	        t.Parallel()
//	        data := build(t)
//	        result, err := data.SUT.DoSomething(data.State.Input)
//	        data.Assert(t, data.SUT, data.State, result, err)
//	    })
//	}
//
// Each test name reflects any alternative combination, e.g.:
//   "Test Alternative #0_1_MyCase"
//
// If no alternatives are defined, names match the registered `TestName` values.
func (ts *TestsBuilder[SUT, STATE, ASSERT]) Tests() iter.Seq2[string, func(t *testing.T) TestData[SUT, STATE, ASSERT]] {
	return func(yield func(string, func(t *testing.T) TestData[SUT, STATE, ASSERT]) bool) {
		testSets := ts.GenerateTestSets()
		for _, tset := range testSets {
			testCases := tset.TestCases
			for i, curcase := range testCases {
				build := func(t *testing.T) TestData[SUT, STATE, ASSERT] {
					var sut SUT
					var state STATE

					for j, testcase := range testCases {
						if builder := testcase.StateBuilder; builder != nil {
							builder(t, &sut, &state)
						}

						if j < i {
							continue
						}

						if testcase.SpecificBuilder != nil {
							testcase.SpecificBuilder(t, &sut, &state)
						}

						break
					}

					return TestData[SUT, STATE, ASSERT]{
						SUT:    sut,
						State:  state,
						Assert: curcase.Assertion,
					}
				}

				testName := curcase.TestName
				if tset.TestSetName != "" {
					testName = fmt.Sprintf("Test Alternative #%s_%s", tset.TestSetName, curcase.TestName)
				}
				if !yield(testName, build) {
					return
				}
			}

		}
	}
}
