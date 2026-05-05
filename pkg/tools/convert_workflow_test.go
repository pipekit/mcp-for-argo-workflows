package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertWorkflowTool(t *testing.T) {
	tool := ConvertWorkflowTool()

	assert.Equal(t, "convert_workflow", tool.Name)
	assert.Contains(t, tool.Description, "Convert")
	assert.Contains(t, tool.Description, "deprecated")
}

func TestConvertWorkflowHandler(t *testing.T) {
	handler := ConvertWorkflowHandler()

	//nolint:govet // Field order optimized for readability in tests
	tests := []struct {
		checkManifest func(t *testing.T, manifest string)
		input         ConvertWorkflowInput
		name          string
		errContains   string
		wantKind      string
		wantChanges   []string
		wantWarnings  []string
		wantErr       bool
	}{
		{
			name: "successful workflow conversion - no changes needed",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			wantErr:  false,
			wantKind: "Workflow",
		},
		{
			name: "successful cronworkflow conversion - schedule to schedules",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-cron
spec:
  schedule: "0 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			wantErr:     false,
			wantKind:    "CronWorkflow",
			wantChanges: []string{"Migrated spec.schedule to spec.schedules array"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "schedules:")
				assert.Contains(t, manifest, "- 0 * * * *")
			},
		},
		{
			name: "cronworkflow with schedules already set - no migration",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-cron
spec:
  schedules:
  - "0 * * * *"
  - "30 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			wantErr:  false,
			wantKind: "CronWorkflow",
			// No schedule migration change expected since schedules is already set
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "schedules:")
			},
		},
		{
			name: "workflow template conversion",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: my-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			wantErr:  false,
			wantKind: "WorkflowTemplate",
		},
		{
			name: "cluster workflow template conversion",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: my-cluster-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			wantErr:  false,
			wantKind: "ClusterWorkflowTemplate",
		},
		{
			name: "workflow with deprecated mutex - migrates to mutexes",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: mutex-workflow
spec:
  entrypoint: main
  synchronization:
    mutex:
      name: my-mutex
  templates:
  - name: main
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "Workflow",
			wantChanges: []string{"mutex to spec.synchronization.mutexes"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "mutexes:")
				assert.Contains(t, manifest, "my-mutex")
			},
		},
		{
			name: "workflow with deprecated semaphore - migrates to semaphores",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: semaphore-workflow
spec:
  entrypoint: main
  synchronization:
    semaphore:
      configMapKeyRef:
        name: my-config
        key: workflow
  templates:
  - name: main
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "Workflow",
			wantChanges: []string{"semaphore to spec.synchronization.semaphores"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "semaphores:")
				assert.Contains(t, manifest, "my-config")
			},
		},
		{
			name: "template-level synchronization conversion",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: template-sync-workflow
spec:
  entrypoint: main
  templates:
  - name: main
    synchronization:
      mutex:
        name: template-mutex
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "Workflow",
			wantChanges: []string{"mutex to spec.synchronization.mutexes"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "mutexes:")
				assert.Contains(t, manifest, "template-mutex")
			},
		},
		{
			name: "workflowtemplate with deprecated mutex - migrates to mutexes",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: mutex-template
spec:
  entrypoint: main
  synchronization:
    mutex:
      name: template-mutex
  templates:
  - name: main
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "WorkflowTemplate",
			wantChanges: []string{"mutex to spec.synchronization.mutexes"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "mutexes:")
				assert.Contains(t, manifest, "template-mutex")
			},
		},
		{
			name: "clusterworkflowtemplate with deprecated semaphore - migrates to semaphores",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: semaphore-cluster-template
spec:
  entrypoint: main
  synchronization:
    semaphore:
      configMapKeyRef:
        name: cluster-config
        key: workflow
  templates:
  - name: main
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "ClusterWorkflowTemplate",
			wantChanges: []string{"semaphore to spec.synchronization.semaphores"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "semaphores:")
				assert.Contains(t, manifest, "cluster-config")
			},
		},
		{
			name: "workflowtemplate with template-level synchronization",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-level-sync
spec:
  entrypoint: main
  templates:
  - name: main
    synchronization:
      semaphore:
        configMapKeyRef:
          name: my-semaphore
          key: count
    container:
      image: alpine
`,
			},
			wantErr:     false,
			wantKind:    "WorkflowTemplate",
			wantChanges: []string{"semaphore to spec.synchronization.semaphores"},
			checkManifest: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "semaphores:")
				assert.Contains(t, manifest, "my-semaphore")
			},
		},
		{
			name: "json output format",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
`,
				OutputFormat: "json",
			},
			wantErr:  false,
			wantKind: "Workflow",
			checkManifest: func(t *testing.T, manifest string) {
				// JSON format should have braces
				assert.True(t, strings.HasPrefix(strings.TrimSpace(manifest), "{"))
				assert.Contains(t, manifest, `"kind": "Workflow"`)
			},
		},
		{
			name: "error - empty manifest",
			input: ConvertWorkflowInput{
				Manifest: "",
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "error - whitespace only manifest",
			input: ConvertWorkflowInput{
				Manifest: "   \n\t  ",
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "error - invalid yaml",
			input: ConvertWorkflowInput{
				Manifest: "this: is: not: valid: yaml:",
			},
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "error - unsupported kind",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			},
			wantErr:     true,
			errContains: "unsupported manifest kind",
		},
		{
			name: "error - invalid output format",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
`,
				OutputFormat: "xml",
			},
			wantErr:     true,
			errContains: "invalid output format",
		},
		{
			name: "error - manifest too large",
			input: ConvertWorkflowInput{
				Manifest: strings.Repeat("x", 1<<20+1), // > 1 MiB
			},
			wantErr:     true,
			errContains: "manifest too large",
		},
		{
			name: "workflow without explicit kind defaults to Workflow",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
metadata:
  name: implicit-workflow
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
`,
			},
			wantErr:  false,
			wantKind: "Workflow",
		},
		{
			name: "cronworkflow missing concurrencyPolicy warning",
			input: ConvertWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: no-policy-cron
spec:
  schedules:
  - "0 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
`,
			},
			wantErr:      false,
			wantKind:     "CronWorkflow",
			wantWarnings: []string{"No concurrencyPolicy set"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, output, err := handler(context.Background(), nil, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
				assert.Nil(t, output)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, output)

			// Check kind
			if tt.wantKind != "" {
				assert.Equal(t, tt.wantKind, output.Kind)
			}

			// Check changes
			if len(tt.wantChanges) > 0 {
				for _, wantChange := range tt.wantChanges {
					found := false
					for _, change := range output.Changes {
						if strings.Contains(change, wantChange) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected change %q not found in %v", wantChange, output.Changes)
				}
			}

			// Check warnings
			if len(tt.wantWarnings) > 0 {
				for _, wantWarning := range tt.wantWarnings {
					found := false
					for _, warning := range output.Warnings {
						if strings.Contains(warning, wantWarning) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected warning %q not found in %v", wantWarning, output.Warnings)
				}
			}

			// Check manifest output
			assert.NotEmpty(t, output.Manifest)
			if tt.checkManifest != nil {
				tt.checkManifest(t, output.Manifest)
			}

			// Check result content
			require.Len(t, result.Content, 1)
			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "expected TextContent")
			assert.NotEmpty(t, textContent.Text)
			assert.Contains(t, textContent.Text, tt.wantKind)
		})
	}
}

func TestConvertWorkflowHandler_OutputFormats(t *testing.T) {
	handler := ConvertWorkflowHandler()

	manifest := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
`

	//nolint:govet // Field order optimized for readability in tests
	tests := []struct {
		checkContent   func(t *testing.T, manifest string)
		format         string
		expectedFormat string
	}{
		{
			format:         "",
			expectedFormat: "yaml",
			checkContent: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "apiVersion:")
				assert.Contains(t, manifest, "kind:")
			},
		},
		{
			format:         "yaml",
			expectedFormat: "yaml",
			checkContent: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "apiVersion:")
			},
		},
		{
			format:         "YAML",
			expectedFormat: "yaml",
			checkContent: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, "apiVersion:")
			},
		},
		{
			format:         "json",
			expectedFormat: "json",
			checkContent: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, `"apiVersion"`)
				assert.Contains(t, manifest, `"kind"`)
			},
		},
		{
			format:         "JSON",
			expectedFormat: "json",
			checkContent: func(t *testing.T, manifest string) {
				assert.Contains(t, manifest, `"apiVersion"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run("format_"+tt.format, func(t *testing.T) {
			input := ConvertWorkflowInput{
				Manifest:     manifest,
				OutputFormat: tt.format,
			}

			_, output, err := handler(context.Background(), nil, input)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedFormat, output.Format)
			tt.checkContent(t, output.Manifest)
		})
	}
}

func TestConvertCronWorkflowSpec_ScheduleMigration(t *testing.T) {
	handler := ConvertWorkflowHandler()

	// Test that schedule is properly migrated to schedules
	input := ConvertWorkflowInput{
		Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: migrate-schedule
spec:
  schedule: "*/5 * * * *"
  concurrencyPolicy: Replace
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
`,
	}

	_, output, err := handler(context.Background(), nil, input)
	require.NoError(t, err)

	// Should have migration change
	assert.Contains(t, output.Changes, "Migrated spec.schedule to spec.schedules array")

	// Manifest should contain schedules array
	assert.Contains(t, output.Manifest, "schedules:")
	assert.Contains(t, output.Manifest, "*/5 * * * *")

	// No warnings about concurrencyPolicy since it's set
	for _, warning := range output.Warnings {
		assert.NotContains(t, warning, "concurrencyPolicy")
	}
}
