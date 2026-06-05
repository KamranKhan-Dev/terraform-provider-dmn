package parse

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidFile(t *testing.T) {
	data, err := os.ReadFile("../model/testdata/simple.dmn")
	require.NoError(t, err)

	m, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "1.3", m.Version)

	dec, err := m.FindDecision("Network Lookup")
	require.NoError(t, err)
	assert.Equal(t, "dec_net", dec.ID)

	// resolvable by id too
	dec2, err := m.FindDecision("dec_net")
	require.NoError(t, err)
	assert.Equal(t, dec, dec2)
}

func TestParseUnknownDecision(t *testing.T) {
	data, _ := os.ReadFile("../model/testdata/simple.dmn")
	m, _ := Parse(data)
	_, err := m.FindDecision("Nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Network Lookup") // lists available
}

func TestParseRejectsNonDMN(t *testing.T) {
	_, err := Parse([]byte(`<foo/>`))
	require.Error(t, err)
}

func TestParseRejectsUnknownHitPolicy(t *testing.T) {
	xml := `<definitions xmlns="https://www.omg.org/spec/DMN/20191111/MODEL/" name="x">
	  <decision id="d" name="D"><decisionTable hitPolicy="BOGUS">
	    <output id="o" name="r"/></decisionTable></decision></definitions>`
	_, err := Parse([]byte(xml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
}
