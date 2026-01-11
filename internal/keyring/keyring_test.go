package keyring

import (
	"errors"
	"testing"
)

// MockStore implements Store for testing
type MockStore struct {
	data   map[string]string
	getErr error
	setErr error
	delErr error
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]string),
	}
}

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

func (m *MockStore) Set(service, key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	k := service + ":" + key
	m.data[k] = value
	return nil
}

func (m *MockStore) Delete(service, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	k := service + ":" + key
	delete(m.data, k)
	return nil
}

func TestStore_SetAndGet(t *testing.T) {
	store := NewMockStore()

	err := store.Set("pub", "secret_key", "test-secret-123")
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	got, err := store.Get("pub", "secret_key")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "test-secret-123" {
		t.Errorf("Get() = %q, want %q", got, "test-secret-123")
	}
}

func TestStore_GetNotFound(t *testing.T) {
	store := NewMockStore()

	_, err := store.Get("pub", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewMockStore()

	// Set a value
	_ = store.Set("pub", "secret_key", "to-delete")

	// Delete it
	err := store.Delete("pub", "secret_key")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	// Verify it's gone
	_, err = store.Get("pub", "secret_key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestStore_OverwriteValue(t *testing.T) {
	store := NewMockStore()

	_ = store.Set("pub", "key", "value1")
	_ = store.Set("pub", "key", "value2")

	got, _ := store.Get("pub", "key")
	if got != "value2" {
		t.Errorf("Get() = %q, want %q after overwrite", got, "value2")
	}
}

func TestSystemStore_ImplementsInterface(t *testing.T) {
	// Compile-time check that SystemStore implements Store
	var _ Store = (*SystemStore)(nil)
}
