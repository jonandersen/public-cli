package keyring

import (
	"errors"
	"testing"
)

func TestMockStore_ImplementsInterface(t *testing.T) {
	var _ Store = (*MockStore)(nil)
}

func TestMockStore_SetAndGet(t *testing.T) {
	store := NewMockStore()

	err := store.Set(ServiceName, KeySecretKey, "test-secret-123")
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	got, err := store.Get(ServiceName, KeySecretKey)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "test-secret-123" {
		t.Errorf("Get() = %q, want %q", got, "test-secret-123")
	}
}

func TestMockStore_GetNotFound(t *testing.T) {
	store := NewMockStore()

	_, err := store.Get("pub", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestMockStore_Delete(t *testing.T) {
	store := NewMockStore()

	_ = store.Set(ServiceName, KeySecretKey, "to-delete")

	err := store.Delete(ServiceName, KeySecretKey)
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	_, err = store.Get(ServiceName, KeySecretKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestMockStore_OverwriteValue(t *testing.T) {
	store := NewMockStore()

	_ = store.Set("pub", "key", "value1")
	_ = store.Set("pub", "key", "value2")

	got, _ := store.Get("pub", "key")
	if got != "value2" {
		t.Errorf("Get() = %q, want %q after overwrite", got, "value2")
	}
}

func TestMockStore_WithGetError(t *testing.T) {
	testErr := errors.New("get failed")
	store := NewMockStore().WithGetError(testErr)

	_, err := store.Get("pub", "key")
	if !errors.Is(err, testErr) {
		t.Errorf("Get() error = %v, want %v", err, testErr)
	}
}

func TestMockStore_WithSetError(t *testing.T) {
	testErr := errors.New("set failed")
	store := NewMockStore().WithSetError(testErr)

	err := store.Set("pub", "key", "value")
	if !errors.Is(err, testErr) {
		t.Errorf("Set() error = %v, want %v", err, testErr)
	}
}

func TestMockStore_WithDeleteError(t *testing.T) {
	testErr := errors.New("delete failed")
	store := NewMockStore().WithDeleteError(testErr)

	err := store.Delete("pub", "key")
	if !errors.Is(err, testErr) {
		t.Errorf("Delete() error = %v, want %v", err, testErr)
	}
}

func TestMockStore_WithData(t *testing.T) {
	store := NewMockStore().WithData("pub", "key", "preset-value")

	got, err := store.Get("pub", "key")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "preset-value" {
		t.Errorf("Get() = %q, want %q", got, "preset-value")
	}
}

func TestMockStore_ChainedMethods(t *testing.T) {
	store := NewMockStore().
		WithData("pub", "key1", "value1").
		WithData("pub", "key2", "value2")

	got1, _ := store.Get("pub", "key1")
	got2, _ := store.Get("pub", "key2")

	if got1 != "value1" {
		t.Errorf("Get(key1) = %q, want %q", got1, "value1")
	}
	if got2 != "value2" {
		t.Errorf("Get(key2) = %q, want %q", got2, "value2")
	}
}

func TestMockStore_IsolatedByService(t *testing.T) {
	store := NewMockStore()

	_ = store.Set("service1", "key", "value1")
	_ = store.Set("service2", "key", "value2")

	got1, _ := store.Get("service1", "key")
	got2, _ := store.Get("service2", "key")

	if got1 != "value1" {
		t.Errorf("Get(service1, key) = %q, want %q", got1, "value1")
	}
	if got2 != "value2" {
		t.Errorf("Get(service2, key) = %q, want %q", got2, "value2")
	}
}
