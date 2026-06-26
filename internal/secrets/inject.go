package secrets

import (
	"context"
	"fmt"

	pb "github.com/brazier/brazier/proto/gen"
)

// InjectSecrets resolves each secret name in dispatch.Spec.Secrets using backend
// and appends them as EnvVar entries in dispatch.Env. The secret's name becomes
// the env var key; its value is the plaintext secret value.
// Returns an error if any secret cannot be resolved.
func InjectSecrets(ctx context.Context, dispatch *pb.JobDispatch, backend SecretBackend) error {
	if dispatch.Spec == nil {
		return nil
	}
	for _, name := range dispatch.Spec.Secrets {
		val, err := backend.Get(ctx, name)
		if err != nil {
			return fmt.Errorf("resolve secret %q: %w", name, err)
		}
		dispatch.Env = append(dispatch.Env, &pb.EnvVar{Key: name, Value: val})
	}
	return nil
}
