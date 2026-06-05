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
	default:
		return nil, fmt.Errorf("hit policy %q not implemented yet", hp)
	}
}

func entries(dt *model.DecisionTable, matched []matchedRule) []any {
	out := make([]any, 0, len(matched))
	for _, m := range matched {
		out = append(out, shapeEntry(dt, m.outputs))
	}
	return out
}
