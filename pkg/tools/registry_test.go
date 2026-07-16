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
	readOnlySet := toRegistrarSet(toolRegistrars(true))
	fullSet := toRegistrarSet(toolRegistrars(false))

	assert.NotEmpty(t, specs)
	assert.Equal(t, len(specs), len(fullSet), "write mode should register all tools")

	for _, spec := range specs {
		registrarPtr := reflect.ValueOf(spec.register).Pointer()
		_, inFull := fullSet[registrarPtr]
		_, inReadOnly := readOnlySet[registrarPtr]
		readonlyHint := isReadOnlyTool(spec.tool())

		assert.True(t, inFull)
		assert.Equal(t, readonlyHint, inReadOnly)
	}
}
