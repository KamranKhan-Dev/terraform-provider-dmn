package provider

import (
	"context"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/eval"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/parse"
	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/tfbridge"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*EvaluateFunction)(nil)

type EvaluateFunction struct{}

func NewEvaluateFunction() function.Function { return &EvaluateFunction{} }

func (f *EvaluateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "evaluate"
}

func (f *EvaluateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Evaluate a DMN decision",
		Description: "Parses a DMN definitions document and evaluates the named decision (resolving its DRG dependencies), returning a list of result entries.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "definitions",
				Description: "The DMN XML document, e.g. file(\"decision.dmn\").",
			},
			function.StringParameter{
				Name:        "decision_name",
				Description: "The name (or id) of the decision to evaluate.",
			},
			function.DynamicParameter{
				Name:        "inputs",
				Description: "An object of input variables keyed by the decision's input names.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *EvaluateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var definitions, decisionName string
	var inputs types.Dynamic

	resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &definitions, &decisionName, &inputs))
	if resp.Error != nil {
		return
	}

	goInputs, err := tfbridge.DynamicToGo(ctx, inputs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(2, "invalid inputs: "+err.Error())
		return
	}
	inputMap, ok := goInputs.(map[string]any)
	if goInputs == nil {
		inputMap = map[string]any{}
	} else if !ok {
		resp.Error = function.NewArgumentFuncError(2, "inputs must be an object")
		return
	}

	m, err := parse.Parse([]byte(definitions))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	result, err := eval.New().Evaluate(m, decisionName, inputMap)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	out, err := tfbridge.GoToAttr(ctx, result)
	if err != nil {
		resp.Error = function.NewFuncError("encoding result: " + err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, types.DynamicValue(out)))
}
