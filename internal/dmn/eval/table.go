package eval

import (
	"fmt"
	"strings"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
)

// matchedRule captures a rule that fired and its evaluated output entry.
type matchedRule struct {
	index   int
	outputs map[string]any // keyed by output name (or "" for a single unnamed output)
}

// evalTable evaluates a decision table against ctx and returns the result entries
// after applying the hit policy. Each entry is an object (map[string]any) of named
// outputs, except a single unnamed output yields the bare value (see shapeEntry).
func (e *Engine) evalTable(dt *model.DecisionTable, ctx feel.Scope) ([]any, error) {
	// 1. Evaluate each input expression once.
	inputVals := make([]any, len(dt.Inputs))
	for i, in := range dt.Inputs {
		v, err := e.Feel.EvalExpression(in.InputExpression.Text, ctx)
		if err != nil {
			return nil, fmt.Errorf("input %d (%q): %w", i+1, in.InputExpression.Text, err)
		}
		inputVals[i] = v
	}

	// 2. Find matching rules.
	var matched []matchedRule
	for ri, rule := range dt.Rules {
		ok, err := e.ruleMatches(rule, inputVals, ctx)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", ri+1, err)
		}
		if !ok {
			continue
		}
		outs, err := e.evalOutputs(rule, dt, ctx)
		if err != nil {
			return nil, fmt.Errorf("rule %d outputs: %w", ri+1, err)
		}
		matched = append(matched, matchedRule{index: ri, outputs: outs})
	}

	// 3. Apply hit policy.
	return e.applyHitPolicy(dt, matched)
}

func (e *Engine) ruleMatches(rule model.Rule, inputVals []any, ctx feel.Scope) (bool, error) {
	for ci, entry := range rule.InputEntries {
		if ci >= len(inputVals) {
			return false, fmt.Errorf("rule has more input entries than the table has inputs")
		}
		match, err := e.Feel.EvalUnaryTest(inputVals[ci], entry.Text, ctx)
		if err != nil {
			return false, fmt.Errorf("input entry %d (%q): %w", ci+1, entry.Text, err)
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func (e *Engine) evalOutputs(rule model.Rule, dt *model.DecisionTable, ctx feel.Scope) (map[string]any, error) {
	outs := make(map[string]any, len(dt.Outputs))
	for oi, out := range dt.Outputs {
		if oi >= len(rule.OutputEntries) {
			return nil, fmt.Errorf("rule has %d output entries but table has %d outputs", len(rule.OutputEntries), len(dt.Outputs))
		}
		v, err := e.Feel.EvalExpression(rule.OutputEntries[oi].Text, ctx)
		if err != nil {
			return nil, fmt.Errorf("output %q (%q): %w", out.Name, rule.OutputEntries[oi].Text, err)
		}
		outs[out.Name] = v
	}
	return outs, nil
}

// shapeEntry converts a matched rule's output map into a result entry: a bare
// value when the table has a single unnamed output, otherwise the object itself.
func shapeEntry(dt *model.DecisionTable, outputs map[string]any) any {
	if len(dt.Outputs) == 1 && dt.Outputs[0].Name == "" {
		return outputs[""]
	}
	return outputs
}

func (e *Engine) applyHitPolicy(dt *model.DecisionTable, matched []matchedRule) ([]any, error) {
	hp := strings.ToUpper(strings.TrimSpace(dt.HitPolicy))
	if hp == "" {
		hp = "UNIQUE"
	}
	switch hp {
	case "UNIQUE":
		if len(matched) > 1 {
			return nil, fmt.Errorf("UNIQUE hit policy matched %d rules; exactly one expected", len(matched))
		}
		return entries(dt, matched), nil

	case "ANY":
		if len(matched) > 1 {
			first := matched[0].outputs
			for _, m := range matched[1:] {
				if !sameOutputs(first, m.outputs) {
					return nil, fmt.Errorf("ANY hit policy matched %d rules with differing outputs", len(matched))
				}
			}
			matched = matched[:1]
		}
		return entries(dt, matched), nil

	case "FIRST":
		if len(matched) > 1 {
			matched = matched[:1] // matched is already in rule order
		}
		return entries(dt, matched), nil

	case "COLLECT":
		agg := strings.ToUpper(strings.TrimSpace(dt.Aggregation))
		if agg == "" {
			return entries(dt, matched), nil
		}
		return aggregate(dt, matched, agg)

	case "RULE ORDER", "PRIORITY", "OUTPUT ORDER":
		// v1: PRIORITY/OUTPUT ORDER require output allowedValues ordering (deferred);
		// they currently behave as RULE ORDER. Documented limitation.
		return entries(dt, matched), nil

	default:
		return nil, fmt.Errorf("unsupported hit policy %q", hp)
	}
}

func sameOutputs(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		if bv, ok := b[k]; !ok || fmt.Sprintf("%v", av) != fmt.Sprintf("%v", bv) {
			return false
		}
	}
	return true
}

// aggregate applies a COLLECT aggregator to the single output column and returns
// a one-element list holding the scalar result.
func aggregate(dt *model.DecisionTable, matched []matchedRule, agg string) ([]any, error) {
	if agg == "COUNT" {
		return []any{float64(len(matched))}, nil
	}
	if len(dt.Outputs) != 1 {
		return nil, fmt.Errorf("COLLECT %s requires exactly one output column, found %d", agg, len(dt.Outputs))
	}
	name := dt.Outputs[0].Name
	nums := make([]float64, 0, len(matched))
	for _, m := range matched {
		f, ok := toFloat(m.outputs[name])
		if !ok {
			return nil, fmt.Errorf("COLLECT %s requires numeric outputs, got %T", agg, m.outputs[name])
		}
		nums = append(nums, f)
	}
	if len(nums) == 0 {
		return []any{}, nil
	}
	switch agg {
	case "SUM":
		s := 0.0
		for _, n := range nums {
			s += n
		}
		return []any{s}, nil
	case "MIN":
		m := nums[0]
		for _, n := range nums[1:] {
			if n < m {
				m = n
			}
		}
		return []any{m}, nil
	case "MAX":
		m := nums[0]
		for _, n := range nums[1:] {
			if n > m {
				m = n
			}
		}
		return []any{m}, nil
	default:
		return nil, fmt.Errorf("unknown COLLECT aggregation %q", agg)
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func entries(dt *model.DecisionTable, matched []matchedRule) []any {
	out := make([]any, 0, len(matched))
	for _, m := range matched {
		out = append(out, shapeEntry(dt, m.outputs))
	}
	return out
}
