package feel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvalExpression(t *testing.T) {
	e := New()
	v, err := e.EvalExpression("foo + bar", Scope{"foo": 5, "bar": 7})
	require.NoError(t, err)
	assert.Equal(t, float64(12), v)
}

func TestEvalExpressionReferencesContext(t *testing.T) {
	e := New()
	v, err := e.EvalExpression("tier", Scope{"tier": "gold"})
	require.NoError(t, err)
	assert.Equal(t, "gold", v)
}

func TestUnaryWildcard(t *testing.T) {
	e := New()
	for _, test := range []string{"", "-", "  "} {
		match, err := e.EvalUnaryTest("anything", test, nil)
		require.NoError(t, err)
		assert.True(t, match, "wildcard %q should match", test)
	}
}

func TestUnaryStringEquality(t *testing.T) {
	e := New()
	match, err := e.EvalUnaryTest("prod", `"prod"`, nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest("dev", `"prod"`, nil)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestUnaryNumericEquality(t *testing.T) {
	e := New()
	match, err := e.EvalUnaryTest(float64(21), "21", nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest(float64(22), "21", nil)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestUnaryExplicitEquality(t *testing.T) {
	e := New()
	match, err := e.EvalUnaryTest("prod", `= "prod"`, nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest("prod", `!= "prod"`, nil)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = e.EvalUnaryTest("dev", `!= "prod"`, nil)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestUnaryComparisonOperators(t *testing.T) {
	e := New()
	cases := []struct {
		test  string
		input float64
		want  bool
	}{
		{"> 8", 10, true},
		{"> 8", 8, false},
		{">= 100", 100, true},
		{">= 100", 99, false},
		{"< 5", 4, true},
		{"< 5", 5, false},
		{"<= 5", 5, true},
		{"> 8, <= 5", 4, true},  // disjunction
		{"> 8, <= 5", 7, false}, // neither
	}
	for _, c := range cases {
		match, err := e.EvalUnaryTest(c.input, c.test, nil)
		require.NoError(t, err, c.test)
		assert.Equal(t, c.want, match, "test %q input %v", c.test, c.input)
	}
}

func TestUnaryList(t *testing.T) {
	e := New()
	match, err := e.EvalUnaryTest("b", `"a","b","c"`, nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest("z", `"a","b","c"`, nil)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = e.EvalUnaryTest(float64(2), `1,2,3`, nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest(float64(9), `1,2,3`, nil)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestUnaryListWithCommaInString(t *testing.T) {
	e := New()
	// Top-level comma split must not break a comma inside a quoted string.
	match, err := e.EvalUnaryTest("a,b", `"a,b","c"`, nil)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestUnaryRange(t *testing.T) {
	e := New()
	cases := []struct {
		test  string
		input float64
		want  bool
	}{
		{"[1..10]", 5, true},
		{"[1..10]", 1, true},
		{"[1..10]", 10, true},
		{"[1..10]", 11, false},
		{"[1..10]", 0, false},
		{"(1..10)", 1, false}, // open lower bound excludes 1
		{"(1..10)", 10, false},
		{"[1..10)", 10, false},
	}
	for _, c := range cases {
		match, err := e.EvalUnaryTest(c.input, c.test, nil)
		require.NoError(t, err, c.test)
		assert.Equal(t, c.want, match, "test %q input %v", c.test, c.input)
	}
}

func TestUnaryNot(t *testing.T) {
	e := New()
	match, err := e.EvalUnaryTest("y", `not("x")`, nil)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = e.EvalUnaryTest("x", `not("x")`, nil)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestUnaryReferencesContext(t *testing.T) {
	e := New()
	// A bare value test can reference other inputs in scope.
	match, err := e.EvalUnaryTest(float64(10), "threshold", Scope{"threshold": float64(10)})
	require.NoError(t, err)
	assert.True(t, match)
}

func TestUnaryRangeNonNumericInputErrors(t *testing.T) {
	e := New()
	// A numeric range against a non-numeric input is a type mismatch, not a
	// silent non-match.
	_, err := e.EvalUnaryTest("abc", "[1..10]", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "numeric")
}

func TestUnaryListEscapedQuote(t *testing.T) {
	e := New()
	// The comma splitter must not break on a comma inside a string that also
	// contains an escaped quote.
	match, err := e.EvalUnaryTest(`a"b`, `"a\"b","c"`, nil)
	require.NoError(t, err)
	assert.True(t, match)
}
