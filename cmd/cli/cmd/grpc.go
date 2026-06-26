package cmd

import (
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// dial opens an insecure gRPC connection to addr.
// apiKey is sent as a header on every RPC via per-call credentials.
func dial(addr, _ string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// authCtx attaches the API key as gRPC metadata on the outgoing context.
func authCtx(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+key)
}

// resolveAPIKey returns the API key from the --api-key flag or BRAZIER_API_KEY env var.
func resolveAPIKey() string {
	if apiKey != "" {
		return apiKey
	}
	return os.Getenv("BRAZIER_API_KEY")
}
