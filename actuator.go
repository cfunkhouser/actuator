package actuator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/sirupsen/logrus"
)

// Action which can be taken when an alertmanager payload is sent to a handler.
type Action interface {
	// Act on the alertmanager webhook payload.
	Act(AlermanagerWebhookPayload) error
}

// LogAction does nothing but log an alertmanager payload.
type LogAction struct{}

func (*LogAction) Act(payload AlermanagerWebhookPayload) error {
	logrus.WithFields(logrus.Fields{
		"alertmanager": payload.Alertmanager,
		"alert_count":  len(payload.Alerts),
	}).Info("Acting on payload")
	return nil
}

type commandAction struct {
	command []string
}

func (a *commandAction) Act(payload AlermanagerWebhookPayload) error {
	cmds := strings.Join(a.command, " ")
	logrus.WithFields(logrus.Fields{
		"alertmanager": payload.Alertmanager,
		"alert_count":  len(payload.Alerts),
		"command":      cmds,
	}).Info("executing command in response to payload")

	cmd := exec.Command(a.command[0], a.command[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"command": cmds,
		}).Errorf("failed: %q", out)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"command": cmds,
	}).Infof("success: %q", out)
	return nil
}

// CommandAction executes a command in response to an alertmanager payload.
func CommandAction(command string) (Action, error) {
	cmd, err := shlex.Split(command, true)
	if err != nil {
		return nil, err
	}
	return &commandAction{
		command: cmd,
	}, nil
}

func fail(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func failContentType(w http.ResponseWriter, r *http.Request) (fails bool) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		logrus.WithFields(logrus.Fields{
			"client":           r.RemoteAddr,
			"bad_content_type": ct,
		}).Warn("got unexpected Content-Type from client")
		fail(w, fmt.Sprintf("Not sure what to do with Content-Type %q", ct), http.StatusBadRequest)
		fails = true
	}
	return
}

type actionHandler struct {
	path   string
	token  string
	action Action
}

func (h *actionHandler) failPath(w http.ResponseWriter, r *http.Request) (fails bool) {
	if rp := r.URL.Path; rp != h.path {
		logrus.WithFields(logrus.Fields{
			"client":   r.RemoteAddr,
			"bad_path": rp,
		}).Error("asked to handle unknown path")
		fail(w, fmt.Sprintf("Not sure how to handle path %q", rp), http.StatusNotFound)
		fails = true
	}
	return
}

func (h *actionHandler) failToken(w http.ResponseWriter, r *http.Request) bool {
	if h.token == "" {
		logrus.WithField("client", r.RemoteAddr).Info("no Authorization required")
		return false
	}
	ah := r.Header.Get("Authorization")
	if ah == "" || !strings.HasPrefix(ah, "Bearer ") {
		logrus.WithField("client", r.RemoteAddr).Warn("no valid Authorization header from client")
		fail(w, "No valid Authorization header provided", http.StatusUnauthorized)
		return true
	}
	if token := strings.TrimSpace(strings.TrimPrefix(ah, "Bearer ")); token != h.token {
		logrus.WithFields(logrus.Fields{
			"client":    r.RemoteAddr,
			"bad_token": token,
		}).Warn("bad Authorization token from client")
		fail(w, "Invalid Token Provided", http.StatusUnauthorized)
		return true
	}
	return false
}

func (h *actionHandler) decodePayload(w http.ResponseWriter, r *http.Request) (*AlermanagerWebhookPayload, bool) {
	var payload AlermanagerWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logrus.WithError(err).WithField("client", r.RemoteAddr).Warn("failed receiving payload from client")
		fail(w, fmt.Sprintf("Something unexpected happened reading the payload: %v", err), http.StatusInternalServerError)
		return nil, false
	}
	if v := payload.Version; v != "4" {
		logrus.WithFields(logrus.Fields{
			"client":      r.RemoteAddr,
			"bad_version": v,
		}).Warn("got unexpected payload version from client")
		fail(w, fmt.Sprintf("Unexpected payload version %q from Alertmanager", v), http.StatusBadRequest)
		return nil, false
	}
	return &payload, true
}

func (h *actionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.failPath(w, r) {
		return
	}
	if h.failToken(w, r) {
		return
	}
	if failContentType(w, r) {
		return
	}
	defer r.Body.Close()
	if payload, ok := h.decodePayload(w, r); ok {
		// Give the handlers a copy, so that if there are ever multiple handlers,
		// one can't mess things up for the others.
		if err := h.action.Act(*payload); err != nil {
			fail(w, fmt.Sprintf("Action failed: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// Option which may be passed to Handle.
type Option func(*actionHandler)

// WithToken option sets a Bearer token to protect calls to the handler.
func WithToken(token string) Option {
	return func(h *actionHandler) {
		h.token = token
	}
}

// Handle alertmanager webhook payloads at path, with the provided Action, using
// the provided opts (if any).
func Handle(path string, with Action, opts ...Option) http.Handler {
	h := &actionHandler{
		path:   path,
		action: with,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

type defaultHandler struct{}

func (h *defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp := r.URL.Path
	logrus.WithFields(logrus.Fields{
		"client": r.RemoteAddr,
		"path":   rp,
	}).Info("request matched no configured handlers")
	fail(w, fmt.Sprintf("Not found: %v", rp), http.StatusNotFound)
}

var dh = &defaultHandler{}

// DefaultHandler for any HTTP requests not matching a configured alertmanager
// webhook handler.
func DefaultHandler() http.Handler {
	return dh
}
