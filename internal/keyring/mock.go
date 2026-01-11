package keyring

// MockStore implements Store for testing. It stores secrets in memory
// and can be configured to return errors for testing error handling.
type MockStore struct {
	data   map[string]string
	getErr error
	setErr error
	delErr error
}

// NewMockStore creates a new mock keyring store for testing.
func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]string),
	}
}

// Get retrieves a secret from the mock store.
func (m *MockStore) Get(service, key string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	k := service + ":" + key
	v, ok := m.data[k]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// Set stores a secret in the mock store.
func (m *MockStore) Set(service, key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	k := service + ":" + key
	m.data[k] = value
	return nil
}

// Delete removes a secret from the mock store.
func (m *MockStore) Delete(service, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	k := service + ":" + key
	delete(m.data, k)
	return nil
}

// WithGetError configures the mock to return an error on Get calls.
func (m *MockStore) WithGetError(err error) *MockStore {
	m.getErr = err
	return m
}

// WithSetError configures the mock to return an error on Set calls.
func (m *MockStore) WithSetError(err error) *MockStore {
	m.setErr = err
	return m
}

// WithDeleteError configures the mock to return an error on Delete calls.
func (m *MockStore) WithDeleteError(err error) *MockStore {
	m.delErr = err
	return m
}

// WithData pre-populates the mock store with a secret.
func (m *MockStore) WithData(service, key, value string) *MockStore {
	k := service + ":" + key
	m.data[k] = value
	return m
}
