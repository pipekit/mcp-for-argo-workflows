package tools

import (
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func registrarName(r ToolRegistrar) string {
	full := runtime.FuncForPC(reflect.ValueOf(r).Pointer()).Name()
	if idx := strings.LastIndex(full, "."); idx >= 0 {
		return full[idx+1:]
	}
	return full
}

func registrarNames(registrars []ToolRegistrar) []string {
	names := make([]string, 0, len(registrars))
	for _, r := range registrars {
		names = append(names, registrarName(r))
	}
	return names
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func TestToolSetComposition_ReadOnlyAndWriteCountsAndMembership(t *testing.T) {
	readOnlyExpected := []string{
		"RegisterListWorkflows",
		"RegisterGetWorkflow",
		"RegisterWatchWorkflow",
		"RegisterLogsWorkflow",
		"RegisterWaitWorkflow",
		"RegisterLintWorkflow",
		"RegisterLintWorkflowTemplate",
		"RegisterLintClusterWorkflowTemplate",
		"RegisterLintCronWorkflow",
		"RegisterRenderWorkflowGraph",
		"RegisterRenderManifestGraph",
		"RegisterListWorkflowTemplates",
		"RegisterGetWorkflowTemplate",
		"RegisterListClusterWorkflowTemplates",
		"RegisterGetClusterWorkflowTemplate",
		"RegisterListCronWorkflows",
		"RegisterGetCronWorkflow",
		"RegisterGetWorkflowNode",
		"RegisterConvertWorkflow",
	}

	writeExpected := []string{
		"RegisterSubmitWorkflow",
		"RegisterDeleteWorkflow",
		"RegisterRetryWorkflow",
		"RegisterResubmitWorkflow",
		"RegisterSuspendWorkflow",
		"RegisterResumeWorkflow",
		"RegisterStopWorkflow",
		"RegisterTerminateWorkflow",
		"RegisterCreateWorkflowTemplate",
		"RegisterDeleteWorkflowTemplate",
		"RegisterCreateClusterWorkflowTemplate",
		"RegisterDeleteClusterWorkflowTemplate",
		"RegisterCreateCronWorkflow",
		"RegisterDeleteCronWorkflow",
		"RegisterSuspendCronWorkflow",
		"RegisterResumeCronWorkflow",
		"RegisterDeleteArchivedWorkflow",
		"RegisterResubmitArchivedWorkflow",
		"RegisterRetryArchivedWorkflow",
	}

	readOnlyActual := registrarNames(ReadOnlyTools())
	writeActual := registrarNames(WriteTools())

	assert.Len(t, readOnlyActual, len(readOnlyExpected), "read-only tool count changed")
	assert.Len(t, writeActual, len(writeExpected), "write tool count changed")
	assert.Equal(t, sortedCopy(readOnlyExpected), sortedCopy(readOnlyActual), "read-only tool membership changed")
	assert.Equal(t, sortedCopy(writeExpected), sortedCopy(writeActual), "write tool membership changed")

	readOnlySet := toSet(readOnlyActual)
	for _, name := range writeActual {
		_, exists := readOnlySet[name]
		assert.False(t, exists, "tool %q should not be in both read-only and write sets", name)
	}

	fullActual := append([]string(nil), readOnlyActual...)
	fullActual = append(fullActual, writeActual...)
	fullExpected := append([]string(nil), readOnlyExpected...)
	fullExpected = append(fullExpected, writeExpected...)
	assert.Len(t, fullActual, len(fullExpected), "full tool count changed")
	assert.Equal(t, sortedCopy(fullExpected), sortedCopy(fullActual), "full tool membership changed")
}
