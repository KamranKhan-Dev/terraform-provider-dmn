// Package parse turns DMN XML bytes into a validated, indexed Model.
package parse

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/dmn/model"
)

// Model wraps parsed Definitions with version info and lookup indexes.
type Model struct {
	Defs    *model.Definitions
	Version string // "1.2".."1.5"

	decByID   map[string]*model.Decision
	decByName map[string]*model.Decision
	inByID    map[string]*model.InputData
}

// namespaceVersions maps the date-code segment of a DMN MODEL namespace to a version.
var namespaceVersions = map[string]string{
	"20180521": "1.2",
	"20191111": "1.3",
	"20211108": "1.4",
	"20230324": "1.5",
}

var validHitPolicies = map[string]bool{
	"": true, "UNIQUE": true, "ANY": true, "FIRST": true, "PRIORITY": true,
	"COLLECT": true, "RULE ORDER": true, "OUTPUT ORDER": true,
}

func Parse(data []byte) (*Model, error) {
	var defs model.Definitions
	if err := xml.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("parsing DMN XML: %w", err)
	}
	if defs.XMLName.Local != "definitions" {
		return nil, fmt.Errorf("not a DMN document: root element is %q, expected \"definitions\"", defs.XMLName.Local)
	}

	version, err := detectVersion(defs.XMLName.Space)
	if err != nil {
		return nil, err
	}

	m := &Model{
		Defs:      &defs,
		Version:   version,
		decByID:   map[string]*model.Decision{},
		decByName: map[string]*model.Decision{},
		inByID:    map[string]*model.InputData{},
	}
	for i := range defs.Decisions {
		d := &defs.Decisions[i]
		m.decByID[d.ID] = d
		if d.Name != "" {
			m.decByName[d.Name] = d
		}
		if d.DecisionTable != nil && !validHitPolicies[strings.ToUpper(d.DecisionTable.HitPolicy)] {
			return nil, fmt.Errorf("decision %q: unknown hit policy %q", d.Name, d.DecisionTable.HitPolicy)
		}
	}
	for i := range defs.InputData {
		in := &defs.InputData[i]
		m.inByID[in.ID] = in
	}
	return m, nil
}

func detectVersion(ns string) (string, error) {
	// Accept http/https; namespace looks like .../spec/DMN/<datecode>/MODEL/
	for code, v := range namespaceVersions {
		if strings.Contains(ns, "/DMN/"+code+"/") {
			return v, nil
		}
	}
	if strings.Contains(ns, "/DMN/") {
		return "", fmt.Errorf("unsupported DMN version (namespace %q); supported: 1.2-1.5", ns)
	}
	return "", fmt.Errorf("not a DMN namespace: %q", ns)
}

// FindDecision resolves a decision by name first, then by id.
func (m *Model) FindDecision(nameOrID string) (*model.Decision, error) {
	if d, ok := m.decByName[nameOrID]; ok {
		return d, nil
	}
	if d, ok := m.decByID[nameOrID]; ok {
		return d, nil
	}
	names := make([]string, 0, len(m.decByName))
	for n := range m.decByName {
		names = append(names, n)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("decision %q not found; available decisions: %s", nameOrID, strings.Join(names, ", "))
}

// DecisionByHref resolves a "#id" reference to a decision.
func (m *Model) DecisionByHref(href string) (*model.Decision, bool) {
	d, ok := m.decByID[strings.TrimPrefix(href, "#")]
	return d, ok
}

// InputDataByHref resolves a "#id" reference to input data.
func (m *Model) InputDataByHref(href string) (*model.InputData, bool) {
	in, ok := m.inByID[strings.TrimPrefix(href, "#")]
	return in, ok
}
