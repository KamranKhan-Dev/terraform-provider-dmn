package model

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalSimpleDMN(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.dmn")
	require.NoError(t, err)

	var defs Definitions
	require.NoError(t, xml.Unmarshal(data, &defs))

	assert.Equal(t, "Networking", defs.Name)
	assert.Equal(t, "https://www.omg.org/spec/DMN/20191111/MODEL/", defs.XMLName.Space)

	require.Len(t, defs.Decisions, 1)
	dec := defs.Decisions[0]
	assert.Equal(t, "Network Lookup", dec.Name)
	require.NotNil(t, dec.DecisionTable)

	dt := dec.DecisionTable
	assert.Equal(t, "UNIQUE", dt.HitPolicy)
	require.Len(t, dt.Inputs, 2)
	assert.Equal(t, "business_unit", dt.Inputs[0].InputExpression.Text)
	require.Len(t, dt.Outputs, 1)
	assert.Equal(t, "subnet_id", dt.Outputs[0].Name)
	require.Len(t, dt.Rules, 2)
	require.Len(t, dt.Rules[0].InputEntries, 2)
	assert.Equal(t, `"payments"`, dt.Rules[0].InputEntries[0].Text)
	assert.Equal(t, `"subnet-aaa"`, dt.Rules[0].OutputEntries[0].Text)
}
