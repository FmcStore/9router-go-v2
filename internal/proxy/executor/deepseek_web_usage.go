package executor

import (
	"encoding/json"
	"strings"
)

// Token accounting for deepseek-web, ported from purujawa06-bot/9router
// (open-sse/executors/deepseek-web.js, 88cfaaf) and sharing the estimator
// gemini_web_usage.go established: the web upstream never reports token
// counts, so the response estimate is what Usage & Analytics sees.
//
// A zero/zero usage row is dropped by the usage recorder — reporting an
// all-zero object (the previous behavior) made every deepseek-web turn
// invisible in analytics. The estimate is coarse but non-zero, which is the
// property the recorder needs.

// deepSeekLastUserText recovers the prompt side of the turn: the last user
// message the request carried, or the session tail the prompt builder chose.
// The accumulator holds no prompt, so the request envelope is the source.
func deepSeekLastUserText(_ *deepSeekStreamAccumulator, req *Request) string {
	if req == nil || len(req.Body) == 0 {
		return ""
	}
	var envelope struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(req.Body, &envelope); err != nil {
		return ""
	}
	for i := len(envelope.Messages) - 1; i >= 0; i-- {
		m := envelope.Messages[i]
		if m.Role != "user" {
			continue
		}
		switch v := m.Content.(type) {
		case string:
			return v
		case []any:
			var sb strings.Builder
			for _, part := range v {
				if pm, ok := part.(map[string]any); ok {
					if text, ok := pm["text"].(string); ok {
						sb.WriteString(text)
					}
				}
			}
			return sb.String()
		}
		return ""
	}
	return ""
}

// deepSeekWebEstimateTokens is geminiWebEstimateTokens under a deepseek name,
// so the two web lanes account on the same scale.
func deepSeekWebEstimateTokens(text string) int {
	return geminiWebEstimateTokens(text)
}

// deepSeekWebBuildUsage renders the estimated token accounting as the OpenAI
// usage object. An all-empty turn still reports ones rather than zeros: the
// point is that the row survives the recorder's zero-drop.
func deepSeekWebBuildUsage(prompt, answer, reasoning string) map[string]any {
	promptTokens := deepSeekWebEstimateTokens(prompt)
	reasoningTokens := deepSeekWebEstimateTokens(reasoning)
	completionTokens := deepSeekWebEstimateTokens(answer) + reasoningTokens
	if completionTokens == 0 {
		completionTokens = 1
	}
	usage := map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
	}
	if reasoningTokens > 0 {
		usage["completion_tokens_details"] = map[string]any{"reasoning_tokens": reasoningTokens}
	}
	return usage
}

