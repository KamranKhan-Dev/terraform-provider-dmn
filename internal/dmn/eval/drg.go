package eval

import (
	"fmt"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/parse"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
)

// Evaluate resolves and evaluates the named decision and its DRG dependencies,
// returning the target decision's result entries (always a list).
func (e *Engine) Evaluate(m *parse.Model, decisionName string, inputs map[string]any) ([]any, error) {
	target, err := m.FindDecision(decisionName)
	if err != nil {
		return nil, err
	}

	// Validate that every required input data item is supplied.
	if err := checkRequiredInputs(m, target, inputs); err != nil {
		return nil, err
	}

	// Base context = caller inputs.
	ctx := feel.Scope{}
	for k, v := range inputs {
		ctx[k] = v
	}

	visiting := map[string]bool{}
	done := map[string]bool{}
	if err := e.resolve(m, target, ctx, visiting, done); err != nil {
		return nil, err
	}
	return e.evalDecision(target, ctx)
}

// resolve evaluates the decision's required decisions depth-first (cycle-detected),
// storing each dependency's primary output into ctx under its variable name.
func (e *Engine) resolve(m *parse.Model, dec *model.Decision, ctx feel.Scope, visiting, done map[string]bool) error {
	if done[dec.ID] {
		return nil
	}
	if visiting[dec.ID] {
		return fmt.Errorf("cycle detected in decision requirements at %q", dec.Name)
	}
	visiting[dec.ID] = true

	for _, req := range dec.InformationReqs {
		if req.RequiredDecision == nil {
			continue
		}
		dep, ok := m.DecisionByHref(req.RequiredDecision.Href)
		if !ok {
			return fmt.Errorf("decision %q requires unknown decision %q", dec.Name, req.RequiredDecision.Href)
		}
		if err := e.resolve(m, dep, ctx, visiting, done); err != nil {
			return err
		}
		result, err := e.evalDecision(dep, ctx)
		if err != nil {
			return fmt.Errorf("evaluating required decision %q: %w", dep.Name, err)
		}
		if err := bindResult(ctx, dep, result); err != nil {
			return err
		}
	}

	visiting[dec.ID] = false
	done[dec.ID] = true
	return nil
}

// evalDecision dispatches on the decision's logic type.
func (e *Engine) evalDecision(dec *model.Decision, ctx feel.Scope) ([]any, error) {
	switch {
	case dec.DecisionTable != nil:
		return e.evalTable(dec.DecisionTable, ctx)
	case dec.LiteralExpression != nil:
		return e.evalLiteral(dec.LiteralExpression, ctx)
	default:
		return nil, fmt.Errorf("decision %q: unsupported decision logic type (only decision tables and literal expressions are supported)", dec.Name)
	}
}

// bindResult exposes a dependency decision's result to downstream decisions under
// its variable name. A single-entry result binds the entry's value; an empty
// result binds nil; multi-entry results bind the list.
//
// When a single result entry is a map containing exactly the variable name key
// (the common case where the table has one output named after the decision
// variable), we unwrap and bind the scalar value directly. This allows
// downstream decisions to reference the variable by name in FEEL expressions.
func bindResult(ctx feel.Scope, dec *model.Decision, result []any) error {
	name := dec.Name
	if dec.Variable != nil && dec.Variable.Name != "" {
		name = dec.Variable.Name
	}
	switch len(result) {
	case 0:
		ctx[name] = nil
	case 1:
		entry := result[0]
		// If the entry is a map whose sole key matches the variable name, unwrap
		// the scalar so downstream expressions can reference it directly.
		if m, ok := entry.(map[string]any); ok {
			if v, hasKey := m[name]; hasKey && len(m) == 1 {
				ctx[name] = v
			} else {
				ctx[name] = entry
			}
		} else {
			ctx[name] = entry
		}
	default:
		ctx[name] = result
	}
	return nil
}

// checkRequiredInputs ensures every inputData transitively required by dec is present.
func checkRequiredInputs(m *parse.Model, dec *model.Decision, inputs map[string]any) error {
	seen := map[string]bool{}
	var walk func(d *model.Decision) error
	walk = func(d *model.Decision) error {
		for _, req := range d.InformationReqs {
			if req.RequiredInput != nil {
				in, ok := m.InputDataByHref(req.RequiredInput.Href)
				if !ok {
					return fmt.Errorf("decision %q requires unknown input data %q", d.Name, req.RequiredInput.Href)
				}
				name := in.Name
				if in.Variable != nil && in.Variable.Name != "" {
					name = in.Variable.Name
				}
				if _, present := inputs[name]; !present {
					return fmt.Errorf("missing required input %q", name)
				}
			}
			if req.RequiredDecision != nil && !seen[req.RequiredDecision.Href] {
				seen[req.RequiredDecision.Href] = true
				if dep, ok := m.DecisionByHref(req.RequiredDecision.Href); ok {
					if err := walk(dep); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	return walk(dec)
}
