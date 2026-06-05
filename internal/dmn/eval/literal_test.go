package eval

import (
	"testing"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvalLiteralExpression(t *testing.T) {
	e := &Engine{Feel: feel.New()}
	lit := &model.LiteralExpression{Text: "a * b"}
	got, err := e.evalLiteral(lit, feel.Scope{"a": float64(6), "b": float64(7)})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, float64(42), got[0])
}
