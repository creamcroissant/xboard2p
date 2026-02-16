package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/creamcroissant/xboard/internal/agentclient"
	"github.com/creamcroissant/xboard/internal/api/requestctx"
	"github.com/creamcroissant/xboard/internal/service"
	"github.com/creamcroissant/xboard/internal/support/i18n"
)

// AdminProtocolHandler handles protocol management via Panel.
type AdminProtocolHandler struct {
	agentHostSvc service.AgentHostService
	i18n         *i18n.Manager
}

// NewAdminProtocolHandler creates a new admin protocol handler.
func NewAdminProtocolHandler(agentHostSvc service.AgentHostService, i18nMgr *i18n.Manager) *AdminProtocolHandler {
	return &AdminProtocolHandler{
		agentHostSvc: agentHostSvc,
		i18n:         i18nMgr,
	}
}

// getAgentClient creates an agent client for the specified host.
func (h *AdminProtocolHandler) getAgentClient(ctx context.Context, hostID int64) (*agentclient.Client, error) {
	host, err := h.agentHostSvc.GetByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("host not found / 节点不存在: %w", err)
	}

	// Build agent URL from host address
	// Assume agent runs on port 8081 by default
	agentURL := fmt.Sprintf("http://%s:8081", host.Host)
	client := agentclient.NewClient(agentURL, host.Token)

	return client, nil
}

// ListConfigs handles GET /api/v1/admin/agent-hosts/{id}/protocols
// Returns all protocol configuration files from an agent.
func (h *AdminProtocolHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.list_configs", "error.bad_request", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.list_configs", "error.not_found", h.i18n)
		return
	}

	configs, err := client.ListConfigs(ctx)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.list_configs", "error.bad_gateway", h.i18n)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": configs,
	})
}

// ListInbounds handles GET /api/v1/admin/agent-hosts/{id}/protocols/inbounds
// Returns only inbound configuration files from an agent.
func (h *AdminProtocolHandler) ListInbounds(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.list_inbounds", "error.bad_request", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.list_inbounds", "error.not_found", h.i18n)
		return
	}

	inbounds, err := client.ListInbounds(ctx)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.list_inbounds", "error.bad_gateway", h.i18n)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": inbounds,
	})
}

// GetConfig handles GET /api/v1/admin/agent-hosts/{id}/protocols/{filename}
// Returns the content of a specific protocol configuration file.
func (h *AdminProtocolHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.get_config", "error.bad_request", h.i18n)
		return
	}

	filename := r.PathValue("filename")
	if filename == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.get_config", "error.missing_fields", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.get_config", "error.not_found", h.i18n)
		return
	}

	content, err := client.GetConfig(ctx, filename)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.get_config", "error.bad_gateway", h.i18n)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

// ApplyProtocolRequest is the request body for applying a protocol configuration.
type ApplyProtocolRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// ApplyConfig handles POST /api/v1/admin/agent-hosts/{id}/protocols
// Applies a protocol configuration to an agent.
func (h *AdminProtocolHandler) ApplyConfig(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_config", "error.bad_request", h.i18n)
		return
	}

	var req ApplyProtocolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_config", "error.bad_request", h.i18n)
		return
	}

	if req.Filename == "" || req.Content == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_config", "error.missing_fields", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.apply_config", "error.not_found", h.i18n)
		return
	}

	if err := client.ApplyConfig(ctx, req.Filename, req.Content); err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.apply_config", "error.bad_gateway", h.i18n)
		return
	}

	RespondSuccessI18n(ctx, w, "success.updated", h.i18n, nil)
}

// ApplyTemplateRequest is the request body for applying a template.
type ApplyTemplateRequest struct {
	Filename string         `json:"filename"`
	Template string         `json:"template"`
	Vars     map[string]any `json:"vars"`
}

// ApplyTemplate handles POST /api/v1/admin/agent-hosts/{id}/protocols/template
// Applies a template-based configuration to an agent.
func (h *AdminProtocolHandler) ApplyTemplate(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_template", "error.bad_request", h.i18n)
		return
	}

	var req ApplyTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_template", "error.bad_request", h.i18n)
		return
	}

	if req.Filename == "" || req.Template == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.apply_template", "error.missing_fields", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.apply_template", "error.not_found", h.i18n)
		return
	}

	if err := client.ApplyTemplate(ctx, req.Filename, req.Template, req.Vars); err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.apply_template", "error.bad_gateway", h.i18n)
		return
	}

	RespondSuccessI18n(ctx, w, "success.updated", h.i18n, nil)
}

// DeleteConfig handles DELETE /api/v1/admin/agent-hosts/{id}/protocols/{filename}
// Removes a protocol configuration from an agent.
func (h *AdminProtocolHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.delete_config", "error.bad_request", h.i18n)
		return
	}

	filename := r.PathValue("filename")
	if filename == "" {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.delete_config", "error.missing_fields", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.delete_config", "error.not_found", h.i18n)
		return
	}

	if err := client.DeleteConfig(ctx, filename); err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.delete_config", "error.bad_gateway", h.i18n)
		return
	}

	RespondSuccessI18n(ctx, w, "success.deleted", h.i18n, nil)
}

// ServiceStatus handles GET /api/v1/admin/agent-hosts/{id}/service/status
// Returns the service status from an agent.
func (h *AdminProtocolHandler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.service_status", "error.bad_request", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.service_status", "error.not_found", h.i18n)
		return
	}

	running, err := client.ServiceStatus(ctx)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.service_status", "error.bad_gateway", h.i18n)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"running": running,
	})
}

// ReloadService handles POST /api/v1/admin/agent-hosts/{id}/service/reload
// Reloads the sing-box service on an agent.
func (h *AdminProtocolHandler) ReloadService(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.service_reload", "error.bad_request", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.service_reload", "error.not_found", h.i18n)
		return
	}

	if err := client.ReloadService(ctx); err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusBadGateway, "admin.protocol.service_reload", "error.bad_gateway", h.i18n)
		return
	}

	RespondSuccessI18n(ctx, w, "success.updated", h.i18n, nil)
}

// AgentHealth handles GET /api/v1/admin/agent-hosts/{id}/health
// Checks if an agent is reachable.
func (h *AdminProtocolHandler) AgentHealth(w http.ResponseWriter, r *http.Request) {
	hostID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		RespondErrorI18nAction(r.Context(), w, http.StatusBadRequest, "admin.protocol.health", "error.bad_request", h.i18n)
		return
	}

	ctx := r.Context()
	client, err := h.getAgentClient(ctx, hostID)
	if err != nil {
		RespondErrorI18nAction(ctx, w, http.StatusNotFound, "admin.protocol.health", "error.not_found", h.i18n)
		return
	}

	if err := client.Health(ctx); err != nil {
		msgKey := "error.bad_gateway"
		msg := msgKey
		if h.i18n != nil {
			msg = h.i18n.Translate(requestctx.GetLanguage(ctx), msgKey)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthy": false,
			"error":   msg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"healthy": true,
	})
}
