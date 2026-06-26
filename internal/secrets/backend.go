// Package secrets defines the SecretBackend interface and its implementations.
package secrets

import "context"

// SecretBackend reads, writes, and deletes named secrets.
type SecretBackend interface {
	Get(ctx context.Context, name string) (string, error)
	Set(ctx context.Context, name, value string) error
	Delete(ctx context.Context, name string) error
}
