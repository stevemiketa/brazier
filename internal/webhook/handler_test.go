package webhook_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brazier/brazier/internal/webhook"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func pushBody(repo, ref, sha string) []byte {
	b, _ := json.Marshal(map[string]any{
		"ref":   ref,
		"after": sha,
		"repository": map[string]string{
			"full_name": repo,
		},
	})
	return b
}

func TestPushEvent(t *testing.T) {
	const secret = "test-secret"
	var received []webhook.TriggerEvent
	h := webhook.New(secret, func(e webhook.TriggerEvent) { received = append(received, e) })

	body := pushBody("org/repo", "refs/heads/main", "abc123")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sign(secret, body))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
	if len(received) != 1 {
		t.Fatalf("got %d events, want 1", len(received))
	}
	ev := received[0]
	if ev.Project != "org/repo" || ev.Branch != "main" || ev.SHA != "abc123" {
		t.Errorf("event = %+v", ev)
	}
}

func TestInvalidSignature(t *testing.T) {
	h := webhook.New("secret", func(webhook.TriggerEvent) {})
	body := pushBody("org/repo", "refs/heads/main", "abc")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", "sha256=badhex")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestTagPush(t *testing.T) {
	const secret = "s"
	var received []webhook.TriggerEvent
	h := webhook.New(secret, func(e webhook.TriggerEvent) { received = append(received, e) })

	body := pushBody("org/repo", "refs/tags/v1.0.0", "deadbeef")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sign(secret, body))

	httptest.NewRecorder()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if len(received) != 1 || received[0].Tag != "v1.0.0" {
		t.Errorf("tag event = %+v", received)
	}
}
