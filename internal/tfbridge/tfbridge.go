// Package tfbridge converts between Terraform framework dynamic values and Go any.
package tfbridge

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DynamicToGo converts a framework dynamic value into a plain Go value
// (map[string]any, []any, string, float64, bool, or nil).
func DynamicToGo(ctx context.Context, d types.Dynamic) (any, error) {
	if d.IsNull() || d.IsUnderlyingValueNull() {
		return nil, nil
	}
	return attrToGo(ctx, d.UnderlyingValue())
}

func attrToGo(ctx context.Context, v attr.Value) (any, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	switch val := v.(type) {
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.NumberValue:
		f, _ := val.ValueBigFloat().Float64()
		return f, nil
	case basetypes.ObjectValue:
		out := make(map[string]any, len(val.Attributes()))
		for k, av := range val.Attributes() {
			gv, err := attrToGo(ctx, av)
			if err != nil {
				return nil, err
			}
			out[k] = gv
		}
		return out, nil
	case basetypes.MapValue:
		out := make(map[string]any, len(val.Elements()))
		for k, av := range val.Elements() {
			gv, err := attrToGo(ctx, av)
			if err != nil {
				return nil, err
			}
			out[k] = gv
		}
		return out, nil
	case basetypes.ListValue:
		return elemsToGo(ctx, val.Elements())
	case basetypes.TupleValue:
		return elemsToGo(ctx, val.Elements())
	case basetypes.SetValue:
		return elemsToGo(ctx, val.Elements())
	case basetypes.DynamicValue:
		return attrToGo(ctx, val.UnderlyingValue())
	default:
		return nil, fmt.Errorf("unsupported terraform type %T", v)
	}
}

func elemsToGo(ctx context.Context, elems []attr.Value) (any, error) {
	out := make([]any, len(elems))
	for i, e := range elems {
		gv, err := attrToGo(ctx, e)
		if err != nil {
			return nil, err
		}
		out[i] = gv
	}
	return out, nil
}

// GoToAttr converts a plain Go value into a framework attr.Value. Lists become
// tuples and maps become objects so heterogeneous shapes are preserved.
func GoToAttr(ctx context.Context, v any) (attr.Value, error) {
	switch val := v.(type) {
	case nil:
		return types.DynamicNull(), nil
	case string:
		return types.StringValue(val), nil
	case bool:
		return types.BoolValue(val), nil
	case float64:
		return types.NumberValue(big.NewFloat(val)), nil
	case int:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case int64:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case []any:
		elems := make([]attr.Value, len(val))
		etypes := make([]attr.Type, len(val))
		for i, e := range val {
			av, err := GoToAttr(ctx, e)
			if err != nil {
				return nil, err
			}
			elems[i] = av
			etypes[i] = av.Type(ctx)
		}
		tup, diags := types.TupleValue(etypes, elems)
		if diags.HasError() {
			return nil, fmt.Errorf("building tuple: %v", diags.Errors())
		}
		return tup, nil
	case map[string]any:
		attrs := make(map[string]attr.Value, len(val))
		atypes := make(map[string]attr.Type, len(val))
		for k, e := range val {
			av, err := GoToAttr(ctx, e)
			if err != nil {
				return nil, err
			}
			attrs[k] = av
			atypes[k] = av.Type(ctx)
		}
		obj, diags := types.ObjectValue(atypes, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("building object: %v", diags.Errors())
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("cannot represent Go value of type %T in Terraform", v)
	}
}
