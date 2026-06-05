package eval

import (
	"testing"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// numTable builds a single-input, single-named-output table whose rules all
// match input >= 0, producing the given numeric outputs in rule order.
func numTable(hp, agg string, outs ...float64) *model.DecisionTable {
	dt := &model.DecisionTable{
		HitPolicy:   hp,
		Aggregation: agg,
		Inputs:      []model.InputClause{{InputExpression: model.InputExpression{Text: "x"}}},
		Outputs:     []model.OutputClause{{Name: "v", TypeRef: "number"}},
	}
	for _, o := range outs {
		dt.Rules = append(dt.Rules, model.Rule{
			InputEntries:  []model.UnaryTest{{Text: ">= 0"}},
			OutputEntries: []model.TextExpr{{Text: ftoa(o)}},
		})
	}
	return dt
}

func ftoa(f float64) string {
	switch f {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	}
	return "0"
}

func run(t *testing.T, dt *model.DecisionTable) []any {
	t.Helper()
	e := &Engine{Feel: feel.New()}
	got, err := e.evalTable(dt, feel.Scope{"x": float64(5)})
	require.NoError(t, err)
	return got
}

func vals(entries []any) []any {
	out := make([]any, len(entries))
	for i, e := range entries {
		out[i] = e.(map[string]any)["v"]
	}
	return out
}

func TestFirst(t *testing.T) {
	got := run(t, numTable("FIRST", "", 1, 2, 3))
	require.Len(t, got, 1)
	assert.Equal(t, float64(1), got[0].(map[string]any)["v"])
}

func TestAnyIdentical(t *testing.T) {
	got := run(t, numTable("ANY", "", 2, 2, 2))
	require.Len(t, got, 1)
	assert.Equal(t, float64(2), got[0].(map[string]any)["v"])
}

func TestAnyConflict(t *testing.T) {
	e := &Engine{Feel: feel.New()}
	_, err := e.evalTable(numTable("ANY", "", 1, 2), feel.Scope{"x": float64(5)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANY")
}

func TestRuleOrder(t *testing.T) {
	got := run(t, numTable("RULE ORDER", "", 3, 1, 2))
	assert.Equal(t, []any{float64(3), float64(1), float64(2)}, vals(got))
}

func TestCollect(t *testing.T) {
	got := run(t, numTable("COLLECT", "", 1, 2, 3))
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, vals(got))
}

func TestCollectSum(t *testing.T) {
	got := run(t, numTable("COLLECT", "SUM", 1, 2, 3))
	require.Len(t, got, 1)
	assert.Equal(t, float64(6), got[0])
}

func TestCollectCount(t *testing.T) {
	got := run(t, numTable("COLLECT", "COUNT", 1, 2, 3))
	require.Len(t, got, 1)
	assert.Equal(t, float64(3), got[0])
}

func TestCollectMinMax(t *testing.T) {
	min := run(t, numTable("COLLECT", "MIN", 3, 1, 2))
	assert.Equal(t, float64(1), min[0])
	max := run(t, numTable("COLLECT", "MAX", 3, 1, 2))
	assert.Equal(t, float64(3), max[0])
}
