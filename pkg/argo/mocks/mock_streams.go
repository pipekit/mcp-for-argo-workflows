// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"io"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"google.golang.org/grpc/metadata"
)

// MockWorkflowLogsStream implements workflow.WorkflowService_WorkflowLogsClient for testing.
//
//nolint:govet // Field order optimized for readability over alignment
type MockWorkflowLogsStream struct {
	entries []*workflow.LogEntry
	err     error // Error to return on Recv
	//nolint:containedctx // Required for grpc.ClientStream interface
	ctx   context.Context
	index int
}

// NewMockWorkflowLogsStream creates a new mock logs stream with the given log entries.
func NewMockWorkflowLogsStream(entries []*workflow.LogEntry) *MockWorkflowLogsStream {
	return &MockWorkflowLogsStream{
		entries: entries,
		index:   0,
		ctx:     context.Background(),
	}
}

// NewMockWorkflowLogsStreamWithError creates a mock logs stream that returns an error.
func NewMockWorkflowLogsStreamWithError(err error) *MockWorkflowLogsStream {
	return &MockWorkflowLogsStream{
		err: err,
		ctx: context.Background(),
	}
}

// SetContext sets the context for this stream.
func (m *MockWorkflowLogsStream) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Recv returns the next log entry or io.EOF when done.
func (m *MockWorkflowLogsStream) Recv() (*workflow.LogEntry, error) {
	// Check if we should return an error
	if m.err != nil {
		return nil, m.err
	}

	// Check context cancellation
	if m.ctx.Err() != nil {
		return nil, m.ctx.Err()
	}

	// Return EOF if no more entries
	if m.index >= len(m.entries) {
		return nil, io.EOF
	}

	entry := m.entries[m.index]
	m.index++
	return entry, nil
}

// Header implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) Header() (metadata.MD, error) {
	return nil, nil
}

// Trailer implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) Trailer() metadata.MD {
	return nil
}

// CloseSend implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) CloseSend() error {
	return nil
}

// Context implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) Context() context.Context {
	return m.ctx
}

// SendMsg implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg implements grpc.ClientStream.
func (m *MockWorkflowLogsStream) RecvMsg(_ interface{}) error {
	return nil
}

// Ensure MockWorkflowLogsStream implements the interface.
var _ workflow.WorkflowService_WorkflowLogsClient = (*MockWorkflowLogsStream)(nil)

// MockWatchWorkflowsStream implements workflow.WorkflowService_WatchWorkflowsClient for testing.
//
//nolint:govet // Field order optimized for readability over alignment
type MockWatchWorkflowsStream struct {
	events []*workflow.WorkflowWatchEvent
	err    error // Error to return on Recv
	//nolint:containedctx // Required for grpc.ClientStream interface
	ctx   context.Context
	index int
}

// NewMockWatchWorkflowsStream creates a new mock watch stream with the given events.
func NewMockWatchWorkflowsStream(events []*workflow.WorkflowWatchEvent) *MockWatchWorkflowsStream {
	return &MockWatchWorkflowsStream{
		events: events,
		index:  0,
		ctx:    context.Background(),
	}
}

// NewMockWatchWorkflowsStreamWithError creates a mock watch stream that returns an error.
func NewMockWatchWorkflowsStreamWithError(err error) *MockWatchWorkflowsStream {
	return &MockWatchWorkflowsStream{
		err: err,
		ctx: context.Background(),
	}
}

// SetContext sets the context for this stream.
func (m *MockWatchWorkflowsStream) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Recv returns the next watch event or io.EOF when done.
func (m *MockWatchWorkflowsStream) Recv() (*workflow.WorkflowWatchEvent, error) {
	// Check if we should return an error
	if m.err != nil {
		return nil, m.err
	}

	// Check context cancellation
	if m.ctx.Err() != nil {
		return nil, m.ctx.Err()
	}

	// Return EOF if no more events
	if m.index >= len(m.events) {
		return nil, io.EOF
	}

	event := m.events[m.index]
	m.index++
	return event, nil
}

// Header implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) Header() (metadata.MD, error) {
	return nil, nil
}

// Trailer implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) Trailer() metadata.MD {
	return nil
}

// CloseSend implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) CloseSend() error {
	return nil
}

// Context implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) Context() context.Context {
	return m.ctx
}

// SendMsg implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg implements grpc.ClientStream.
func (m *MockWatchWorkflowsStream) RecvMsg(_ interface{}) error {
	return nil
}

// Ensure MockWatchWorkflowsStream implements the interface.
var _ workflow.WorkflowService_WatchWorkflowsClient = (*MockWatchWorkflowsStream)(nil)

// Helper functions for building test data

// NewLogEntry creates a log entry for testing.
func NewLogEntry(podName, content string) *workflow.LogEntry {
	return &workflow.LogEntry{
		PodName: podName,
		Content: content,
	}
}

// NewWatchEvent creates a watch event for testing.
func NewWatchEvent(eventType string, wf *wfv1.Workflow) *workflow.WorkflowWatchEvent {
	return &workflow.WorkflowWatchEvent{
		Type:   eventType,
		Object: wf,
	}
}
