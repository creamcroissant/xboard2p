package handler

import (
	"encoding/json"
	"net/http"

	"github.com/creamcroissant/xboard/internal/api/requestctx"
	"github.com/creamcroissant/xboard/internal/service"
	"github.com/creamcroissant/xboard/internal/support/i18n"
)

type AdminInviteHandler struct {
	invites service.InviteService
	i18n    *i18n.Manager
}

func NewAdminInviteHandler(invites service.InviteService, i18nMgr *i18n.Manager) *AdminInviteHandler {
	return &AdminInviteHandler{invites: invites, i18n: i18nMgr}
}

func (h *AdminInviteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actions like list, generate
	action := adminInviteActionPath(r.URL.Path)
	switch {
	case action == "/generate" && r.Method == http.MethodPost:
		h.handleGenerate(w, r)
	case action == "/fetch" && (r.Method == http.MethodGet || r.Method == http.MethodPost):
		h.handleFetch(w, r)
	default:
		respondNotImplemented(w, "admin.invite", r)
	}
}

func (h *AdminInviteHandler) handleFetch(w http.ResponseWriter, r *http.Request) {
	claims := requestctx.AdminFromContext(r.Context())
	if claims.ID == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusUnauthorized, "admin.invite.fetch", "error.unauthorized", h.i18n)
		return
	}

	limit := 20
	offset := 0
	// Parse pagination if needed (for now simplified or assume default)
	// If payload is JSON, decode it.

	codes, total, err := h.invites.Fetch(r.Context(), limit, offset)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusInternalServerError, "admin.invite.fetch", "error.internal_server_error", h.i18n)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data":  codes,
		"total": total,
	})
}

type generateInviteRequest struct {
	Count    int   `json:"count"`
	Limit    int64 `json:"limit"`
	ExpireAt int64 `json:"expire_at"`
}

func (h *AdminInviteHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	claims := requestctx.AdminFromContext(r.Context())
	if claims.ID == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusUnauthorized, "admin.invite.generate", "error.unauthorized", h.i18n)
		return
	}

	var req generateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.invite.generate", "error.bad_request", h.i18n)
		return
	}

	// Assuming 0 userID for admin generated codes or use admin's ID if available/needed
	// Since AdminFromContext returns ID as string (UUID?), we need to resolve it to int64 if required by InviteService.
	// But InviteService.GenerateBatch takes userID int64.
	// For now, let's pass 0 for system/admin generated.
	err := h.invites.GenerateBatch(r.Context(), req.Count, req.Limit, req.ExpireAt, 0)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusInternalServerError, "admin.invite.generate", "error.internal_server_error", h.i18n)
		return
	}

	RespondSuccessI18n(r.Context(), w, "success.created", h.i18n, nil)
}

func adminInviteActionPath(fullPath string) string {
	return adminActionPath(fullPath)
}