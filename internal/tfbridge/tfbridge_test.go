package tfbridge

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bigF(f float64) *big.Float { return big.NewFloat(f) }

func TestObjectToGo(t *testing.T) {
	ctx := context.Background()
	obj, diags := types.ObjectValue(
		map[string]attr.Type{"business_unit": types.StringType, "spend": types.NumberType},
		map[string]attr.Value{
			"business_unit": types.StringValue("payments"),
			"spend":         types.NumberValue(bigF(150)),
		},
	)
	require.False(t, diags.HasError())

	got, err := DynamicToGo(ctx, types.DynamicValue(obj))
	require.NoError(t, err)
	m := got.(map[string]any)
	assert.Equal(t, "payments", m["business_unit"])
	assert.Equal(t, float64(150), m["spend"])
}

func TestGoToAttr_ListOfObjects(t *testing.T) {
	ctx := context.Background()
	in := []any{map[string]any{"subnet_id": "subnet-aaa"}}
	v, err := GoToAttr(ctx, in)
	require.NoError(t, err)

	back, err := DynamicToGo(ctx, types.DynamicValue(v))
	require.NoError(t, err)
	list := back.([]any)
	require.Len(t, list, 1)
	assert.Equal(t, "subnet-aaa", list[0].(map[string]any)["subnet_id"])
}

func TestGoToAttr_Scalar(t *testing.T) {
	ctx := context.Background()
	v, err := GoToAttr(ctx, float64(6))
	require.NoError(t, err)
	back, err := DynamicToGo(ctx, types.DynamicValue(v))
	require.NoError(t, err)
	assert.Equal(t, float64(6), back)
}
