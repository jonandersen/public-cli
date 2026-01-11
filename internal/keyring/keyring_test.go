package keyring

import (
	"errors"
	"testing"
)

func TestSystemStore_ImplementsInterface(t *testing.T) {
	// Compile-time check that SystemStore implements Store
	var _ Store = (*SystemStore)(nil)
}

func TestEnvStore_ImplementsInterface(t *testing.T) {
	// Compile-time check that EnvStore implements Store
	var _ Store = (*EnvStore)(nil)
}

func TestEnvStore_GetFromEnvVar(t *testing.T) {
	mock := NewMockStore()
	store := NewEnvStore(mock)

	// Set env var
	t.Setenv(EnvSecretKey, "env-secret-123")

	// Should get from env var, not underlying store
	got, err := store.Get(ServiceName, KeySecretKey)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "env-secret-123" {
		t.Errorf("Get() = %q, want %q", got, "env-secret-123")
	}
}

func TestEnvStore_FallbackToUnderlying(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set(ServiceName, KeySecretKey, "keyring-secret")
	store := NewEnvStore(mock)

	// No env var set, should fall back to underlying store
	got, err := store.Get(ServiceName, KeySecretKey)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "keyring-secret" {
		t.Errorf("Get() = %q, want %q", got, "keyring-secret")
	}
}

func TestEnvStore_EnvVarOnlyForSecretKey(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set(ServiceName, "other_key", "other-value")
	store := NewEnvStore(mock)

	// Env var only affects secret_key lookups
	t.Setenv(EnvSecretKey, "env-secret")

	// Other keys should not use env var
	got, err := store.Get(ServiceName, "other_key")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "other-value" {
		t.Errorf("Get() = %q, want %q", got, "other-value")
	}
}

func TestEnvStore_SetPassesThrough(t *testing.T) {
	mock := NewMockStore()
	store := NewEnvStore(mock)

	err := store.Set(ServiceName, KeySecretKey, "new-secret")
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	// Verify it was stored in underlying store
	got, _ := mock.Get(ServiceName, KeySecretKey)
	if got != "new-secret" {
		t.Errorf("underlying Get() = %q, want %q", got, "new-secret")
	}
}

func TestEnvStore_DeletePassesThrough(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set(ServiceName, KeySecretKey, "to-delete")
	store := NewEnvStore(mock)

	err := store.Delete(ServiceName, KeySecretKey)
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	// Verify it was deleted from underlying store
	_, err = mock.Get(ServiceName, KeySecretKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("underlying Get() after Delete() error = %v, want ErrNotFound", err)
	}
}
