package templates

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/aumahesh/goprose/internal/intermediate"
)

func TestTemplateManager_Render(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	pv := &intermediate.Program{
		Org:                "acme.com/iotcontroller",
		ModuleName:         "TreeColoring",
		PackageName:        "internal",
		InterfaceName:      "Color",
		ImplementationName: "color",
		Variables: map[string]string{
			"parent":   "string",
			"root":     "string",
			"is_green": "bool",
		},
		InitialState: map[string]interface{}{
			"parent":   "abcdef",
			"root":     "00000",
			"is_green": true,
		},
	}

	tpl, err := NewTemplateManager("../../generatedPackages/", pv)
	assert.Nil(t, err)
	assert.NotNil(t, tpl)

	err = tpl.Render()
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("Error rendering template: %s", err)
	}
}
