package eval

import (
	"fmt"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
)

// evalLiteral evaluates a literal-expression decision and returns a one-element
// list holding the result, matching the always-a-list return shape.
func (e *Engine) evalLiteral(lit *model.LiteralExpression, ctx feel.Scope) ([]any, error) {
	v, err := e.Feel.EvalExpression(lit.Text, ctx)
	if err != nil {
		return nil, fmt.Errorf("literal expression %q: %w", lit.Text, err)
	}
	return []any{v}, nil
}
