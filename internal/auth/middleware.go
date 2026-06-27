package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const contextKeyAPIKey contextKey = "api_key"
const contextKeySession contextKey = "session"

// AuthEnabled controls whether auth middleware enforces credentials.
// Set to false to disable auth (e.g. during local development).
var AuthEnabled = true

// HTTPMiddleware wraps an http.Handler, requiring a valid API key or session
// token in the Authorization header ("Bearer <token>").
func HTTPMiddleware(keys *APIKeyStore, sessions *SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !AuthEnabled {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearer(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()

		// Try API key first.
		if strings.HasPrefix(token, apiKeyPrefix) {
			key, err := keys.Validate(ctx, token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx = context.WithValue(ctx, contextKeyAPIKey, key)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fall back to session token.
		sess, err := sessions.Validate(ctx, token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx = context.WithValue(ctx, contextKeySession, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GRPCUnaryInterceptor validates the Authorization metadata on every unary RPC.
func GRPCUnaryInterceptor(keys *APIKeyStore, sessions *SessionStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !AuthEnabled {
			return handler(ctx, req)
		}
		if err := validateGRPC(ctx, keys, sessions); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// GRPCStreamInterceptor validates the Authorization metadata on every streaming RPC.
func GRPCStreamInterceptor(keys *APIKeyStore, sessions *SessionStore) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !AuthEnabled {
			return handler(srv, ss)
		}
		if err := validateGRPC(ss.Context(), keys, sessions); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func validateGRPC(ctx context.Context, keys *APIKeyStore, sessions *SessionStore) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization")
	}
	token := extractBearer(vals[0])
	if token == "" {
		return status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	if strings.HasPrefix(token, apiKeyPrefix) {
		if _, err := keys.Validate(ctx, token); err != nil {
			return status.Error(codes.Unauthenticated, "invalid api key")
		}
		return nil
	}
	if _, err := sessions.Validate(ctx, token); err != nil {
		return status.Error(codes.Unauthenticated, "invalid session")
	}
	return nil
}

func extractBearer(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}

// newID generates a random hex ID for records.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
