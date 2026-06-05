package eval

import (
	"os"
	"testing"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/parse"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/feel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoldenBusinessUnitRange(t *testing.T) {
	data, err := os.ReadFile("testdata/business_unit.dmn")
	require.NoError(t, err)
	m, err := parse.Parse(data)
	require.NoError(t, err)
	e := &Engine{Feel: feel.New()}

	got, err := e.Evaluate(m, "Network Lookup", map[string]any{
		"business_unit": "payments", "replicas": float64(25),
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, map[string]any{
		"subnet_id": "snet-pay-large", "vnet_name": "vnet-pay",
	}, got[0])
}
