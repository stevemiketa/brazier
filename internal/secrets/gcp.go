package secrets

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/option"
)

// GCPSecretBackend reads and writes secrets via GCP Secret Manager.
type GCPSecretBackend struct {
	client    *secretmanager.Client
	projectID string
}

// NewGCPSecretBackend creates a GCPSecretBackend using Application Default Credentials.
func NewGCPSecretBackend(ctx context.Context, projectID string, opts ...option.ClientOption) (*GCPSecretBackend, error) {
	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcp secret manager client: %w", err)
	}
	return &GCPSecretBackend{client: client, projectID: projectID}, nil
}

func (g *GCPSecretBackend) secretName(name string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", g.projectID, name)
}

func (g *GCPSecretBackend) Get(ctx context.Context, name string) (string, error) {
	versionName := g.secretName(name) + "/versions/latest"
	resp, err := g.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: versionName,
	})
	if err != nil {
		return "", fmt.Errorf("gcp get secret %s: %w", name, err)
	}
	return string(resp.Payload.Data), nil
}

func (g *GCPSecretBackend) Set(ctx context.Context, name, value string) error {
	parent := fmt.Sprintf("projects/%s", g.projectID)

	// Ensure the secret exists.
	_, err := g.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: name,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	// Ignore "already exists" errors.
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		return fmt.Errorf("gcp create secret %s: %w", name, err)
	}

	_, err = g.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  g.secretName(name),
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	})
	if err != nil {
		return fmt.Errorf("gcp add secret version %s: %w", name, err)
	}
	return nil
}

func (g *GCPSecretBackend) Delete(ctx context.Context, name string) error {
	err := g.client.DeleteSecret(ctx, &secretmanagerpb.DeleteSecretRequest{
		Name: g.secretName(name),
	})
	if err != nil {
		return fmt.Errorf("gcp delete secret %s: %w", name, err)
	}
	return nil
}

func (g *GCPSecretBackend) Close() error { return g.client.Close() }
