// Package feel isolates FEEL expression evaluation behind an interface so the
// underlying engine (currently github.com/pbinitiative/feel, MIT) can be swapped.
//
// The upstream library evaluates FEEL *expressions* well, but does not implement
// DMN decision-table *unary test* semantics correctly: bare value literals are
// returned as-is instead of being compared to the input ("?"), comma-separated
// literal lists evaluate to a constant true, ranges return a RangeValue object
// rather than a membership boolean, and the "-" wildcard is a parse error.
// Additionally, evaluating an expression containing an explicit "?" token hangs
// the parser. This wrapper therefore implements unary-test semantics itself,
// delegating only safe sub-expressions to the library:
//   - comparison operators (< <= > >=) use the library's implicit-"?" unary path
//   - "=", "!=" and bare values are evaluated without "?" and compared in Go
//   - ranges are evaluated to a RangeValue and tested for membership in Go
//   - "-" / empty is a wildcard, and not(...) negates the inner test
package feel

import (
	"fmt"
	"reflect"
	"strings"

	pbfeel "github.com/pbinitiative/feel"
)

// Scope is a set of FEEL variable bindings.
type Scope = map[string]any

// Evaluator evaluates FEEL expressions and decision-table unary tests.
type Evaluator interface {
	// EvalExpression evaluates a FEEL expression against ctx.
	EvalExpression(expr string, ctx Scope) (any, error)
	// EvalUnaryTest evaluates a decision-table input entry against the input
	// value, returning whether the test matches per DMN unary-test semantics.
	EvalUnaryTest(input any, test string, ctx Scope) (bool, error)
}

// New returns the default Evaluator backed by pbinitiative/feel.
func New() Evaluator { return pbEvaluator{} }

type pbEvaluator struct{}

func (pbEvaluator) EvalExpression(expr string, ctx Scope) (any, error) {
	v, err := pbfeel.EvalStringWithScope(expr, toScope(ctx))
	if err != nil {
		return nil, fmt.Errorf("evaluating FEEL expression %q: %w", expr, err)
	}
	return v, nil
}

func (e pbEvaluator) EvalUnaryTest(input any, test string, ctx Scope) (bool, error) {
	t := strings.TrimSpace(test)
	if t == "" || t == "-" {
		return true, nil // wildcard: matches any value
	}
	for _, el := range splitTopLevelCommas(t) {
		match, err := e.matchElement(input, strings.TrimSpace(el), ctx)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil // comma list is disjunction (OR)
		}
	}
	return false, nil
}

// matchElement evaluates a single unary-test element against input.
func (e pbEvaluator) matchElement(input any, el string, ctx Scope) (bool, error) {
	if el == "" {
		return false, nil
	}

	// not(...) negation.
	if strings.HasPrefix(el, "not(") && strings.HasSuffix(el, ")") {
		inner := strings.TrimSpace(el[len("not(") : len(el)-1])
		match, err := e.EvalUnaryTest(input, inner, ctx)
		if err != nil {
			return false, err
		}
		return !match, nil
	}

	// Comparison operators handled by the library's implicit-"?" unary path.
	for _, op := range []string{">=", "<=", ">", "<"} {
		if strings.HasPrefix(el, op) {
			v, err := pbfeel.EvalStringWithScope(el, toScopeWithInput(ctx, input))
			if err != nil {
				return false, fmt.Errorf("evaluating unary test %q: %w", el, err)
			}
			b, ok := v.(bool)
			if !ok {
				return false, fmt.Errorf("comparison test %q did not yield a boolean (got %T)", el, v)
			}
			return b, nil
		}
	}

	// Explicit equality / inequality.
	if strings.HasPrefix(el, "!=") {
		eq, err := e.valueEquals(input, strings.TrimSpace(el[2:]), ctx)
		return !eq, err
	}
	if strings.HasPrefix(el, "=") {
		return e.valueEquals(input, strings.TrimSpace(el[1:]), ctx)
	}

	// Range membership, e.g. [1..10], (0..5], [0..100).
	if isRange(el) {
		v, err := pbfeel.EvalStringWithScope(el, toScope(ctx))
		if err != nil {
			return false, fmt.Errorf("evaluating range %q: %w", el, err)
		}
		rv, ok := v.(*pbfeel.RangeValue)
		if !ok {
			return false, fmt.Errorf("range test %q did not yield a range (got %T)", el, v)
		}
		return rangeContains(rv, input)
	}

	// Bare value: implicit equality against the input.
	return e.valueEquals(input, el, ctx)
}

// valueEquals evaluates expr (without "?") and compares it to input for equality.
func (e pbEvaluator) valueEquals(input any, expr string, ctx Scope) (bool, error) {
	v, err := pbfeel.EvalStringWithScope(expr, toScope(ctx))
	if err != nil {
		return false, fmt.Errorf("evaluating value %q: %w", expr, err)
	}
	return goEqual(input, v), nil
}

// isRange reports whether el looks like a FEEL range literal.
func isRange(el string) bool {
	if len(el) < 2 {
		return false
	}
	open := el[0] == '[' || el[0] == '('
	last := el[len(el)-1]
	closed := last == ']' || last == ')'
	return open && closed && strings.Contains(el, "..")
}

// rangeContains reports whether input falls within the numeric range rv.
// v1 supports numeric ranges only; non-numeric bounds or inputs are an explicit
// error rather than a silent non-match, so users are not given a wrong result.
func rangeContains(rv *pbfeel.RangeValue, input any) (bool, error) {
	start, ok1 := numberToFloat(rv.Start)
	end, ok2 := numberToFloat(rv.End)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("non-numeric ranges are not supported in v1")
	}
	f, ok := toFloat(input)
	if !ok {
		return false, fmt.Errorf("range test requires a numeric input, got %T", input)
	}
	lower := f > start || (!rv.StartOpen && f == start)
	upper := f < end || (!rv.EndOpen && f == end)
	return lower && upper, nil
}

func numberToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case *pbfeel.Number:
		return n.Float64(), true
	case pbfeel.Number:
		return n.Float64(), true
	default:
		return toFloat(v)
	}
}

// goEqual compares two FEEL values for equality, treating numeric kinds uniformly.
func goEqual(a, b any) bool {
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			return af == bf
		}
		return false
	}
	return reflect.DeepEqual(a, b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case *pbfeel.Number:
		return n.Float64(), true
	case pbfeel.Number:
		return n.Float64(), true
	default:
		return 0, false
	}
}

// splitTopLevelCommas splits s on commas that are not nested inside quotes,
// parentheses, or brackets.
func splitTopLevelCommas(s string) []string {
	var parts []string
	var buf strings.Builder
	depth := 0
	inStr := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inStr && escaped:
			// Previous char was a backslash inside a string: this char is literal.
			escaped = false
			buf.WriteByte(c)
		case inStr && c == '\\':
			escaped = true
			buf.WriteByte(c)
		case c == '"':
			inStr = !inStr
			buf.WriteByte(c)
		case inStr:
			buf.WriteByte(c)
		case c == '(' || c == '[':
			depth++
			buf.WriteByte(c)
		case c == ')' || c == ']':
			depth--
			buf.WriteByte(c)
		case c == ',' && depth == 0:
			parts = append(parts, buf.String())
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}
	parts = append(parts, buf.String())
	return parts
}

func toScope(ctx Scope) pbfeel.Scope {
	s := make(pbfeel.Scope, len(ctx))
	for k, v := range ctx {
		s[k] = v
	}
	return s
}

func toScopeWithInput(ctx Scope, input any) pbfeel.Scope {
	s := toScope(ctx)
	s["?"] = input
	return s
}
