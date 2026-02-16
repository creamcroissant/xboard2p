package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/creamcroissant/xboard/internal/api/requestctx"
	"github.com/creamcroissant/xboard/internal/support/i18n"
)

// Helper to respond with JSON
func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Warn("failed to encode response JSON", "error", err)
	}
}

// Helper to respond with translated JSON
func respondJSONTranslated(ctx context.Context, w http.ResponseWriter, status int, payload any, i18nMgr *i18n.Manager) {
	lang := requestctx.GetLanguage(ctx)
	
	// If payload is a map, try to translate values that look like keys?
	// For now, let's just assume the payload structure is handled by the caller or we translate specific fields.
	// A common pattern is to have a Message field.
	
	if m, ok := payload.(map[string]any); ok {
		if msg, ok := m["message"].(string); ok {
			m["message"] = i18nMgr.Translate(lang, msg)
		}
		if msg, ok := m["error"].(string); ok {
			// If error is a key, translate it.
			// But usually error contains dynamic info. 
			// Let's assume for now we pass keys for static errors.
			m["error"] = i18nMgr.Translate(lang, msg)
		}
	}

	respondJSON(w, status, payload)
}

func respondNotImplemented(w http.ResponseWriter, namespace string, r *http.Request) {
	respondNotImplementedI18n(r.Context(), w, namespace, r.Method, r.URL.Path, nil)
}

func respondNotImplementedI18n(ctx context.Context, w http.ResponseWriter, namespace string, method string, path string, i18nMgr *i18n.Manager) {
	lang := requestctx.GetLanguage(ctx)
	message := "error.not_implemented"
	if i18nMgr != nil {
		message = i18nMgr.Translate(lang, message)
	}
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"message":   message,
		"namespace": namespace,
		"method":    method,
		"path":      path,
	})
}

func respondError(w http.ResponseWriter, status int, action string, err error) {
	respondJSON(w, status, map[string]any{
		"error":  err.Error(),
		"action": action,
	})
}

func RespondErrorI18nAction(ctx context.Context, w http.ResponseWriter, status int, action string, key string, i18nMgr *i18n.Manager, args ...interface{}) {
	if key == "" {
		key = action
	}
	lang := requestctx.GetLanguage(ctx)
	var msg string
	if i18nMgr != nil {
		msg = i18nMgr.Translate(lang, key, args...)
	} else {
		msg = key
	}
	resp := map[string]any{
		"error": msg,
	}
	if action != "" {
		resp["action"] = action
	}
	respondJSON(w, status, resp)
}

// New helper for i18n error responses
func RespondErrorI18n(ctx context.Context, w http.ResponseWriter, status int, key string, i18nMgr *i18n.Manager, args ...interface{}) {
	lang := requestctx.GetLanguage(ctx)
	var msg string
	if i18nMgr != nil {
		msg = i18nMgr.Translate(lang, key, args...)
	} else {
		msg = key // Fallback if manager is missing (e.g. in tests)
	}
	respondJSON(w, status, map[string]any{
		"error": msg,
	})
}

// New helper for i18n success responses
func RespondSuccessI18n(ctx context.Context, w http.ResponseWriter, key string, i18nMgr *i18n.Manager, data any) {
	lang := requestctx.GetLanguage(ctx)
	var msg string
	if i18nMgr != nil {
		msg = i18nMgr.Translate(lang, key)
	} else {
		msg = key // Fallback
	}

	resp := map[string]any{
		"message": msg,
	}
	if data != nil {
		resp["data"] = data
	}

	respondJSON(w, http.StatusOK, resp)
}