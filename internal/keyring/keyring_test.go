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
	t.Setenv("PUB_SECRET_KEY", "env-secret-123")

	// Should get from env var, not underlying store
	got, err := store.Get("pub", "secret_key")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "env-secret-123" {
		t.Errorf("Get() = %q, want %q", got, "env-secret-123")
	}
}

func TestEnvStore_FallbackToUnderlying(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set("pub", "secret_key", "keyring-secret")
	store := NewEnvStore(mock)

	// No env var set, should fall back to underlying store
	got, err := store.Get("pub", "secret_key")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "keyring-secret" {
		t.Errorf("Get() = %q, want %q", got, "keyring-secret")
	}
}

func TestEnvStore_EnvVarOnlyForSecretKey(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set("pub", "other_key", "other-value")
	store := NewEnvStore(mock)

	// Env var only affects secret_key lookups
	t.Setenv("PUB_SECRET_KEY", "env-secret")

	// Other keys should not use env var
	got, err := store.Get("pub", "other_key")
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

	err := store.Set("pub", "secret_key", "new-secret")
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	// Verify it was stored in underlying store
	got, _ := mock.Get("pub", "secret_key")
	if got != "new-secret" {
		t.Errorf("underlying Get() = %q, want %q", got, "new-secret")
	}
}

func TestEnvStore_DeletePassesThrough(t *testing.T) {
	mock := NewMockStore()
	_ = mock.Set("pub", "secret_key", "to-delete")
	store := NewEnvStore(mock)

	err := store.Delete("pub", "secret_key")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	// Verify it was deleted from underlying store
	_, err = mock.Get("pub", "secret_key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("underlying Get() after Delete() error = %v, want ErrNotFound", err)
	}
}
