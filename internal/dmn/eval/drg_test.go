package eval

import (
	"os"
	"testing"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/parse"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateDRG(t *testing.T) {
	data, err := os.ReadFile("testdata/drg.dmn")
	require.NoError(t, err)
	m, err := parse.Parse(data)
	require.NoError(t, err)

	e := &Engine{Feel: feel.New()}
	got, err := e.Evaluate(m, "Discount", map[string]any{"spend": float64(150)})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, map[string]any{"discount": float64(20)}, got[0])
}

func TestEvaluateMissingInput(t *testing.T) {
	data, _ := os.ReadFile("testdata/drg.dmn")
	m, _ := parse.Parse(data)
	e := &Engine{Feel: feel.New()}
	_, err := e.Evaluate(m, "Discount", map[string]any{}) // no spend
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spend")
}

func TestEvaluateCycleDetection(t *testing.T) {
	// A requires B, B requires A.
	xml := `<definitions xmlns="https://www.omg.org/spec/DMN/20191111/MODEL/" name="c">
	  <decision id="A" name="A"><variable name="a"/>
	    <informationRequirement><requiredDecision href="#B"/></informationRequirement>
	    <literalExpression><text>1</text></literalExpression></decision>
	  <decision id="B" name="B"><variable name="b"/>
	    <informationRequirement><requiredDecision href="#A"/></informationRequirement>
	    <literalExpression><text>2</text></literalExpression></decision>
	</definitions>`
	m, err := parse.Parse([]byte(xml))
	require.NoError(t, err)
	e := &Engine{Feel: feel.New()}
	_, err = e.Evaluate(m, "A", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}
