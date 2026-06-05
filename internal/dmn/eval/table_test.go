package eval

import (
	"testing"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func twoInputTable(hp string) *model.DecisionTable {
	return &model.DecisionTable{
		HitPolicy: hp,
		Inputs: []model.InputClause{
			{InputExpression: model.InputExpression{Text: "business_unit"}},
			{InputExpression: model.InputExpression{Text: "environment"}},
		},
		Outputs: []model.OutputClause{{Name: "subnet_id"}},
		Rules: []model.Rule{
			{InputEntries: []model.UnaryTest{{Text: `"payments"`}, {Text: `"prod"`}},
				OutputEntries: []model.TextExpr{{Text: `"subnet-aaa"`}}},
			{InputEntries: []model.UnaryTest{{Text: `"payments"`}, {Text: `"dev"`}},
				OutputEntries: []model.TextExpr{{Text: `"subnet-bbb"`}}},
		},
	}
}

func TestEvalTableUnique(t *testing.T) {
	e := &Engine{Feel: feel.New()}
	ctx := feel.Scope{"business_unit": "payments", "environment": "prod"}

	got, err := e.evalTable(twoInputTable("UNIQUE"), ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, map[string]any{"subnet_id": "subnet-aaa"}, got[0])
}

func TestEvalTableNoMatchEmptyList(t *testing.T) {
	e := &Engine{Feel: feel.New()}
	ctx := feel.Scope{"business_unit": "payments", "environment": "qa"}
	got, err := e.evalTable(twoInputTable("UNIQUE"), ctx)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestEvalTableUniqueViolation(t *testing.T) {
	dt := twoInputTable("UNIQUE")
	dt.Rules[1].InputEntries[1].Text = `"prod"` // make both rules match prod
	e := &Engine{Feel: feel.New()}
	ctx := feel.Scope{"business_unit": "payments", "environment": "prod"}
	_, err := e.evalTable(dt, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UNIQUE")
}

func TestEvalTableDashMatchesAnything(t *testing.T) {
	dt := twoInputTable("UNIQUE")
	dt.Rules[0].InputEntries[1].Text = "-" // env wildcard
	dt.Rules = dt.Rules[:1]                // keep one rule to stay unique
	e := &Engine{Feel: feel.New()}
	got, err := e.evalTable(dt, feel.Scope{"business_unit": "payments", "environment": "anything"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, map[string]any{"subnet_id": "subnet-aaa"}, got[0])
}
