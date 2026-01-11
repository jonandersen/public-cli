package keyring

// Store provides an interface for secure secret storage.
type Store interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}
