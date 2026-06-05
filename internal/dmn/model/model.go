// Package model holds Go structs mapping the DMN XML schema (versions 1.2-1.5).
// encoding/xml matches on local element names regardless of namespace, so a
// single set of structs reads all supported versions; the namespace is captured
// on Definitions.XMLName for version validation in the parse layer.
package model

import "encoding/xml"

type Definitions struct {
	XMLName   xml.Name         `xml:"definitions"`
	ID        string           `xml:"id,attr"`
	Name      string           `xml:"name,attr"`
	Decisions []Decision       `xml:"decision"`
	InputData []InputData      `xml:"inputData"`
	ItemDefs  []ItemDefinition `xml:"itemDefinition"`
}

type Decision struct {
	ID                string                   `xml:"id,attr"`
	Name              string                   `xml:"name,attr"`
	Variable          *Variable                `xml:"variable"`
	DecisionTable     *DecisionTable           `xml:"decisionTable"`
	LiteralExpression *LiteralExpression       `xml:"literalExpression"`
	InformationReqs   []InformationRequirement `xml:"informationRequirement"`
}

type Variable struct {
	Name    string `xml:"name,attr"`
	TypeRef string `xml:"typeRef,attr"`
}

type InformationRequirement struct {
	RequiredDecision *DMNElementReference `xml:"requiredDecision"`
	RequiredInput    *DMNElementReference `xml:"requiredInput"`
}

// DMNElementReference.Href is like "#someId" (local) referencing a decision or inputData.
type DMNElementReference struct {
	Href string `xml:"href,attr"`
}

type InputData struct {
	ID       string    `xml:"id,attr"`
	Name     string    `xml:"name,attr"`
	Variable *Variable `xml:"variable"`
}

type DecisionTable struct {
	ID          string         `xml:"id,attr"`
	HitPolicy   string         `xml:"hitPolicy,attr"`   // empty => UNIQUE
	Aggregation string         `xml:"aggregation,attr"` // SUM|MIN|MAX|COUNT for COLLECT
	Inputs      []InputClause  `xml:"input"`
	Outputs     []OutputClause `xml:"output"`
	Rules       []Rule         `xml:"rule"`
}

type InputClause struct {
	ID              string          `xml:"id,attr"`
	Label           string          `xml:"label,attr"`
	InputExpression InputExpression `xml:"inputExpression"`
}

type InputExpression struct {
	TypeRef string `xml:"typeRef,attr"`
	Text    string `xml:"text"` // FEEL expression producing the input value
}

type OutputClause struct {
	ID      string `xml:"id,attr"`
	Label   string `xml:"label,attr"`
	Name    string `xml:"name,attr"`
	TypeRef string `xml:"typeRef,attr"`
}

type Rule struct {
	ID            string      `xml:"id,attr"`
	InputEntries  []UnaryTest `xml:"inputEntry"`
	OutputEntries []TextExpr  `xml:"outputEntry"`
}

// UnaryTest is a decision-table input entry (a FEEL unary test against "?").
type UnaryTest struct {
	Text string `xml:"text"`
}

// TextExpr is a FEEL expression cell (output entry or literal expression body).
type TextExpr struct {
	Text string `xml:"text"`
}

type LiteralExpression struct {
	TypeRef string `xml:"typeRef,attr"`
	Text    string `xml:"text"`
}

type ItemDefinition struct {
	ID      string `xml:"id,attr"`
	Name    string `xml:"name,attr"`
	TypeRef string `xml:"typeRef,attr"`
}
