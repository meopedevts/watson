package github

import (
	"context"
	"sync"
)

// MockCall records a single invocation of MockExecutor.
type MockCall struct {
	Name  string
	Args  []string
	Stdin string // only populated for RunWithStdin calls
}

// MockResponse is a pre-configured response for a MockExecutor call.
type MockResponse struct {
	Out []byte
	Err error
}

// MockExecutor implements Executor for tests.
// Responses are consumed in FIFO order.
type MockExecutor struct {
	mu        sync.Mutex
	Calls     []MockCall
	Responses []MockResponse
	idx       int
}

func (m *MockExecutor) next() MockResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.Responses) {
		return MockResponse{}
	}
	r := m.Responses[m.idx]
	m.idx++
	return r
}

func (m *MockExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args})
	m.mu.Unlock()
	r := m.next()
	return r.Out, r.Err
}

func (m *MockExecutor) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args, Stdin: stdin})
	m.mu.Unlock()
	r := m.next()
	return r.Out, r.Err
}
