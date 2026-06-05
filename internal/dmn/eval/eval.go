// Package eval evaluates DMN decisions: decision tables, literal expressions,
// and the decision requirements graph (DRG) connecting them.
package eval

import "github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"

// Engine evaluates DMN decisions using a FEEL evaluator.
type Engine struct {
	Feel feel.Evaluator
}

// New returns an Engine backed by the default FEEL evaluator.
func New() *Engine { return &Engine{Feel: feel.New()} }
