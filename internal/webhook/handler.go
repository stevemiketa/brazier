// Package webhook handles inbound GitHub webhook events.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// TriggerEvent is the parsed, validated payload from a GitHub webhook.
type TriggerEvent struct {
	Project string // repository full name, e.g. "org/repo"
	Branch  string
	Tag     string
	Event   string // GitHub event type: push, pull_request, etc.
	SHA     string // head commit SHA
}

// TriggerFunc is called with a validated TriggerEvent.
type TriggerFunc func(TriggerEvent)

// Handler is an http.Handler for GitHub webhook POST requests.
type Handler struct {
	secret  []byte
	trigger TriggerFunc
}

// New returns a Handler that validates HMAC signatures with secret and
// calls trigger for each valid event.
func New(secret string, trigger TriggerFunc) *Handler {
	return &Handler{secret: []byte(secret), trigger: trigger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	if len(h.secret) > 0 {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !h.validSignature(body, sig) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	ev, err := h.parseEvent(eventType, body)
	if err != nil {
		slog.Warn("webhook: parse event", "type", eventType, "err", err)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	slog.Info("webhook received", "event", ev.Event, "project", ev.Project, "branch", ev.Branch)
	h.trigger(ev)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) validSignature(body []byte, sig string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(sig, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(got, expected)
}

// githubPushPayload is the minimal subset of a GitHub push event we need.
type githubPushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

func (h *Handler) parseEvent(eventType string, body []byte) (TriggerEvent, error) {
	switch eventType {
	case "push":
		var p githubPushPayload
		if err := json.Unmarshal(body, &p); err != nil {
			return TriggerEvent{}, fmt.Errorf("parse push: %w", err)
		}
		ev := TriggerEvent{
			Project: p.Repository.FullName,
			Event:   "push",
			SHA:     p.After,
		}
		ref := p.Ref
		if strings.HasPrefix(ref, "refs/heads/") {
			ev.Branch = strings.TrimPrefix(ref, "refs/heads/")
		} else if strings.HasPrefix(ref, "refs/tags/") {
			ev.Tag = strings.TrimPrefix(ref, "refs/tags/")
		}
		return ev, nil

	case "ping":
		// GitHub sends a ping on webhook creation; acknowledge it.
		return TriggerEvent{}, fmt.Errorf("ping: not a trigger")

	default:
		return TriggerEvent{}, fmt.Errorf("unsupported event type: %s", eventType)
	}
}
