package tools

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func toRegistrarSet(registrars []ToolRegistrar) map[uintptr]struct{} {
	set := make(map[uintptr]struct{}, len(registrars))
	for _, registrar := range registrars {
		set[reflect.ValueOf(registrar).Pointer()] = struct{}{}
	}
	return set
}

func TestToolRegistrarsAreDerivedFromToolAnnotations(t *testing.T) {
	specs := allToolSpecs()
	assert.NotEmpty(t, specs)

	for _, multiContext := range []bool{false, true} {
		readOnlySet := toRegistrarSet(toolRegistrars(true, multiContext))
		fullSet := toRegistrarSet(toolRegistrars(false, multiContext))

		for _, spec := range specs {
			registrarPtr := reflect.ValueOf(spec.register).Pointer()
			_, inFull := fullSet[registrarPtr]
			_, inReadOnly := readOnlySet[registrarPtr]

			if spec.multiContextOnly && !multiContext {
				assert.False(t, inFull, "multi-context-only tool registered without multi-context")
				assert.False(t, inReadOnly, "multi-context-only tool registered without multi-context")
				continue
			}

			assert.True(t, inFull)
			assert.Equal(t, isReadOnlyTool(spec.tool()), inReadOnly)
		}
	}
}

func TestListContextsIsMultiContextOnly(t *testing.T) {
	found := false
	for _, spec := range allToolSpecs() {
		if spec.tool().Name == ListContextsToolName {
			found = true
			assert.True(t, spec.multiContextOnly, "list_contexts must only register when multi-context is enabled")
		}
	}
	assert.True(t, found, "list_contexts must be in the tool registry")
}
