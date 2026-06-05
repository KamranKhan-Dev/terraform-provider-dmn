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

func TestGoToAttr_BoolAndEmptyList(t *testing.T) {
	ctx := context.Background()

	b, err := GoToAttr(ctx, true)
	require.NoError(t, err)
	back, err := DynamicToGo(ctx, types.DynamicValue(b))
	require.NoError(t, err)
	assert.Equal(t, true, back)

	empty, err := GoToAttr(ctx, []any{})
	require.NoError(t, err)
	back, err = DynamicToGo(ctx, types.DynamicValue(empty))
	require.NoError(t, err)
	assert.Equal(t, []any{}, back)
}

func TestDynamicToGo_Null(t *testing.T) {
	ctx := context.Background()
	got, err := DynamicToGo(ctx, types.DynamicNull())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGoToAttr_Nil(t *testing.T) {
	ctx := context.Background()
	v, err := GoToAttr(ctx, nil)
	require.NoError(t, err)
	assert.True(t, v.IsNull())
}

func TestGoToAttr_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	_, err := GoToAttr(ctx, struct{ X int }{X: 1})
	require.Error(t, err)
}
