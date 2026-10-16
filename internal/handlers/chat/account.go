package chat

import (
	"net/http"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/middleware"
)

// Self-service account surface (read-only).
//
// These endpoints are mounted under RequireApiKey like /v1/chat/completions,
// so the caller's own key is the only credential involved: a user queries the
// state of the key they are holding and can never see another key's data,
// because every lookup is keyed off the authenticated request. They exist
// apart from the dashboard's /api/keys/* management endpoints on purpose —
// resold keys must not reach anything that mutates policy.

// HandleAccountUsage responds GET /v1/account/usage.
//
// The payload is the same prompt+completion lifetime sum the quota enforcer
// uses (GetApiKeyTokenUsage), so what the user sees here is exactly what the
// gateway will enforce. An unset quota reads as unlimited rather than 0: a
// 0-quota key is refused upstream before it can ever ask about itself.
func (h *ChatHandler) HandleAccountUsage(w http.ResponseWriter, r *http.Request) {
	key := requestKey(r)
	if key == nil {
		handlerutil.WriteJSONError(w, http.StatusUnauthorized, "API key required.")
		return
	}

	used, err := h.Repo.GetApiKeyTokenUsage(key.Key)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, "failed to read usage")
		return
	}

	resp := map[string]any{
		"key":         handlerutil.MaskAPIKey(key.Key),
		"name":        stringPtrOrEmpty(key.Name),
		"active":      key.IsActive == 1,
		"usedTokens":  used,
		"lastUsedAt":  stringPtrOrEmpty(key.LastUsedAt),
		"expiresAt":   stringPtrOrEmpty(key.ExpiresAt),
		"quotaExceeded": false,
	}

	if key.TokenQuota != nil && *key.TokenQuota > 0 {
		quota := *key.TokenQuota
		remaining := quota - used
		if remaining < 0 {
			remaining = 0
		}
		resp["tokenQuota"] = quota
		resp["remainingTokens"] = remaining
		resp["percentUsed"] = float64(used) / float64(quota) * 100
		resp["quotaExceeded"] = used >= quota
	} else {
		resp["tokenQuota"] = 0
		resp["remainingTokens"] = 0
		resp["percentUsed"] = 0
		resp["unlimited"] = true
	}

	handlerutil.WriteJSON(w, http.StatusOK, resp)
}

// HandleAccountModels responds GET /v1/account/models.
//
// The list is the key's own view of the catalog: the same connection-scoped
// model set /v1/models serves, filtered through this key's allowlist
// (api_key_model_access). A key without an allowlist sees everything its
// gateway offers — identical semantics to dispatch.
func (h *ChatHandler) HandleAccountModels(w http.ResponseWriter, r *http.Request) {
	key := requestKey(r)
	if key == nil {
		handlerutil.WriteJSONError(w, http.StatusUnauthorized, "API key required.")
		return
	}

	result := h.buildModelsListResult(r.Context(), modeListConnected)
	visible, err := h.filterModelsByAccess(key, result.Models)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]map[string]any, 0, len(visible))
	for _, m := range visible {
		out = append(out, map[string]any{"id": m.ID, "ownedBy": m.OwnedBy})
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"key":    handlerutil.MaskAPIKey(key.Key),
		"count":  len(out),
		"models": out,
	})
}

func stringPtrOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// guard the imported shapes this file depends on.
var _ = middleware.ApiKeyContextKey
