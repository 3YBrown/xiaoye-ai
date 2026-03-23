package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type deepSeekChatRequest struct {
	Model          string                 `json:"model"`
	Messages       []deepSeekChatMessage  `json:"messages"`
	Temperature    float64                `json:"temperature,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
}

type deepSeekChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func OptimizePrompt(c *gin.Context) {
	startTime := time.Now()
	userID := c.GetUint64("userID")
	userIDStr := strconv.FormatUint(userID, 10)

	req, err := parseAndValidatePromptOptimizeRequest(c)
	if err != nil {
		resp := gin.H{"error": err.Error()}
		logAPICall("/api/prompt/optimize", req, http.StatusBadRequest, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	requiredCredits := config.GetPromptOptimizeCredits()
	if requiredCredits < 0 {
		requiredCredits = 0
	}
	if err := deductPromptOptimizeCredits(userID, user.Credits, requiredCredits); err != nil {
		resp := gin.H{
			"error":            err.Error(),
			"required_credits": requiredCredits,
			"current_balance":  user.Credits,
		}
		logAPICall("/api/prompt/optimize", req, http.StatusPaymentRequired, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusPaymentRequired, resp)
		return
	}

	candidate, usageResp, modelName, err := optimizeSinglePrompt(req)
	if err != nil {
		if requiredCredits > 0 {
			refundCredits(userID, requiredCredits, "prompt-optimize-request-failed")
		}
		log.Printf("[PromptOptimize] failed [user:%d]: %v", userID, err)
		resp := gin.H{"error": "prompt optimization failed, please retry"}
		logAPICall("/api/prompt/optimize", req, http.StatusBadGateway, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusBadGateway, resp)
		return
	}

	creditsRemaining := user.Credits - requiredCredits
	if creditsRemaining < 0 {
		creditsRemaining = 0
	}

	resp := PromptOptimizeResponse{
		RawPrompt:  req.Prompt,
		Candidates: []PromptOptimizeCandidate{candidate},
		Meta: map[string]interface{}{
			"provider":          "deepseek",
			"model":             modelName,
			"creative_mode":     req.CreativeMode,
			"style":             req.Style,
			"prompt_tokens":     usageResp.Usage.PromptTokens,
			"completion_tokens": usageResp.Usage.CompletionTokens,
			"total_tokens":      usageResp.Usage.TotalTokens,
			"credits_spent":     requiredCredits,
			"credits_remaining": creditsRemaining,
			"latency_ms":        time.Since(startTime).Milliseconds(),
		},
	}
	logAPICall("/api/prompt/optimize", req, http.StatusOK, resp, time.Since(startTime), userIDStr)
	c.JSON(http.StatusOK, resp)
}

func parseAndValidatePromptOptimizeRequest(c *gin.Context) (PromptOptimizeRequest, error) {
	var req PromptOptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, fmt.Errorf("invalid request: %w", err)
	}

	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		return req, errors.New("prompt is required")
	}
	req.CreativeMode = normalizeCreativeMode(req.CreativeMode)
	req.Style = normalizePromptStyle(req.Style)
	return req, nil
}

func deductPromptOptimizeCredits(userID uint64, currentCredits, requiredCredits int) error {
	if requiredCredits <= 0 {
		return nil
	}
	if currentCredits < requiredCredits {
		return errors.New("钻石不足")
	}

	updateResult := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, requiredCredits).
		Updates(map[string]interface{}{
			"credits":    gorm.Expr("credits - ?", requiredCredits),
			"updated_at": time.Now(),
		})
	if updateResult.Error != nil || updateResult.RowsAffected == 0 {
		return errors.New("钻石不足或扣费失败")
	}

	if err := recordCreditTransaction(
		db.DB,
		userID,
		-requiredCredits,
		CreditTxTypePromptOptimizeCost,
		"prompt_optimize",
		"",
		"prompt optimization",
	); err != nil {
		refundCredits(userID, requiredCredits, "prompt-optimize-ledger-write-failed")
		return errors.New("记录钻石流水失败")
	}
	return nil
}

func optimizeSinglePrompt(req PromptOptimizeRequest) (PromptOptimizeCandidate, deepSeekChatResponse, string, error) {
	apiKey := strings.TrimSpace(config.GetDeepSeekAPIKey())
	if apiKey == "" {
		return PromptOptimizeCandidate{}, deepSeekChatResponse{}, "", errors.New("DEEPSEEK_API_KEY is not configured")
	}

	modelName := strings.TrimSpace(config.GetDeepSeekModel())
	if modelName == "" {
		modelName = "deepseek-chat"
	}

	requestBody := deepSeekChatRequest{
		Model:       modelName,
		Messages:    buildOptimizeMessages(req),
		Temperature: 0.35,
		MaxTokens:   4096,
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
		Stream: false,
	}

	respBytes, err := callDeepSeekAPI(requestBody, apiKey)
	if err != nil {
		return PromptOptimizeCandidate{}, deepSeekChatResponse{}, modelName, err
	}

	var parsed deepSeekChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return PromptOptimizeCandidate{}, deepSeekChatResponse{}, modelName, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return PromptOptimizeCandidate{}, parsed, modelName, errors.New("deepseek returned empty choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	candidate, err := parseOptimizedCandidate(content)
	if err != nil {
		return PromptOptimizeCandidate{}, parsed, modelName, err
	}
	return candidate, parsed, modelName, nil
}

func callDeepSeekAPI(requestBody deepSeekChatRequest, apiKey string) ([]byte, error) {
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := buildDeepSeekEndpoint(config.GetDeepSeekBaseURL())
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if httpResp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("deepseek status=%d body=%s", httpResp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}

func buildOptimizeMessages(req PromptOptimizeRequest) []deepSeekChatMessage {
	outputCount := getIntParam(req.CurrentParams, "outputCount", 1)
	imageType := getStringParam(req.CurrentParams, "imageType")
	ecommerceType := getStringParam(req.CurrentParams, "ecommerceType")

	systemPrompt := buildUniversalOptimizeSystemPrompt(
		req.CreativeMode,
		req.Style,
		outputCount,
		imageType,
		ecommerceType,
	)

	payload := map[string]interface{}{
		"prompt":         req.Prompt,
		"creative_mode":  req.CreativeMode,
		"style":          req.Style,
		"target_model":   strings.TrimSpace(req.TargetModel),
		"current_params": req.CurrentParams,
	}
	if req.CreativeMode == "ecommerce" {
		payload["output_count"] = outputCount
		if imageType != "" {
			payload["image_type"] = imageType
		}
		if ecommerceType != "" {
			payload["ecommerce_type"] = ecommerceType
		}
	}
	payloadBytes, _ := json.Marshal(payload)

	return []deepSeekChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(payloadBytes)},
	}
}

func buildUniversalOptimizeSystemPrompt(creativeMode, style string, outputCount int, imageType, ecommerceType string) string {
	modeGuides := map[string]string{
		"image":     "图像生成：补全主体、环境、构图、光影、色彩、材质、风格与质量描述。",
		"video":     "视频生成：补全动作轨迹、镜头语言、时序变化、节奏、转场与画面连贯性。",
		"ecommerce": "电商生成：突出卖点、材质与功能表达，强调产品一致性与转化导向措辞。",
	}
	styleGuides := map[string]string{
		"balanced":   "均衡：优先稳定可控，同时保留一定表现力。",
		"creative":   "创意：增强想象力、画面张力和风格表达，但不偏离用户核心意图。",
		"detail":     "细节：补足可执行细节和约束条件，减少歧义。",
		"commercial": "商业：强调质感、品牌调性、卖点表达与转化友好性。",
	}

	modeText, ok := modeGuides[creativeMode]
	if !ok {
		modeText = modeGuides["image"]
	}
	styleText, ok := styleGuides[style]
	if !ok {
		styleText = styleGuides["balanced"]
	}

	lines := []string{
		"你是资深 AI 提示词优化专家，负责把用户原始想法改写成更清晰、更可执行、更稳定的高质量提示词。",
		"你面对的用户想法多种多样，可能很短、很口语、很抽象，或包含混合语言与碎片化信息。",
		"核心原则：保留用户核心意图，不擅自改题，不删减关键限制条件，不引入与需求冲突的新设定。",
		"语言规则：无论用户输入何种语言，最终输出的 prompt 与 reason 必须为中文；必要时先将原意准确转写为中文，再优化。",
		fmt.Sprintf("任务类型：%s。%s", creativeMode, modeText),
		fmt.Sprintf("优化风格：%s。%s", style, styleText),
		"优化方法：",
		"1. 先识别用户明确约束（主体、风格、场景、构图、镜头、尺寸、品牌、禁用项等），这些约束必须保留。",
		"2. 对缺失信息做最小必要补全，使提示词可直接执行，避免过度脑补。",
		"3. 将描述组织为自然、连贯、可被模型稳定理解的句子，减少堆砌和重复。",
		"4. 若原提示词已足够完善，只做轻量润色和结构优化，不强行扩写。",
		"5. 避免空泛词（如“好看”“高级”）单独出现，尽量转为可执行描述。",
		"6. 不输出免责声明、对话语气或教学解释。",
		"必须仅返回一个 JSON 对象，禁止返回 markdown 代码块或额外解释文本。",
		`格式固定为：{"prompt":"...","reason":"..."}`,
		"prompt 要求：",
		"- 单条可直接投喂模型的优化结果。",
		"- 语义完整、结构清晰、可执行、通用性强。",
		"- 与用户需求一致，不偏题。",
		"reason 要求：一句话说明你主要优化了哪些点（简洁，不超过30字）。",
	}

	if creativeMode == "ecommerce" {
		if outputCount < 1 {
			outputCount = 1
		}
		lines = append(lines,
			fmt.Sprintf("电商组图硬性要求：必须在 prompt 中明确写出“生成 %d 张商品图（组图）”。", outputCount),
			"电商组图硬性要求：若为多张组图，需在同一条 prompt 中按序说明每张图的用途/角度/场景，不能写成只生成1张。",
		)
		if imageType != "" {
			lines = append(lines, fmt.Sprintf("补充上下文：用户选择的图片类型是“%s”。", imageType))
		}
		if ecommerceType != "" {
			lines = append(lines, fmt.Sprintf("补充上下文：用户选择的平台是“%s”。", ecommerceType))
		}
	}

	return strings.Join(lines, "\n")
}

func getStringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	val, ok := params[key]
	if !ok || val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func getIntParam(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	val, ok := params[key]
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int32:
		if v > 0 {
			return int(v)
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float32:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func parseOptimizedCandidate(content string) (PromptOptimizeCandidate, error) {
	candidate := PromptOptimizeCandidate{
		ID:    "1",
		Title: "优化结果",
	}

	raw := stripCodeFence(strings.TrimSpace(content))
	if raw == "" {
		return candidate, errors.New("empty model response")
	}
	if firstObj := extractFirstJSONObject(raw); firstObj != "" {
		raw = firstObj
	}

	var one struct {
		Prompt string `json:"prompt"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &one); err == nil && strings.TrimSpace(one.Prompt) != "" {
		candidate.Prompt = strings.TrimSpace(one.Prompt)
		candidate.Reason = strings.TrimSpace(one.Reason)
		if candidate.Reason == "" {
			candidate.Reason = "已按原始意图完成结构化优化。"
		}
		return candidate, nil
	}

	var wrapped struct {
		Candidates []PromptOptimizeCandidate `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapped); err == nil && len(wrapped.Candidates) > 0 {
		first := wrapped.Candidates[0]
		first.Prompt = strings.TrimSpace(first.Prompt)
		if first.Prompt != "" {
			candidate.Prompt = first.Prompt
			firstReason := strings.TrimSpace(first.Reason)
			if firstReason == "" {
				firstReason = "已按原始意图完成结构化优化。"
			}
			candidate.Reason = firstReason
			return candidate, nil
		}
	}

	fallback := normalizeFallbackPrompt(content)
	if fallback == "" {
		return candidate, errors.New("model response does not contain usable prompt")
	}
	candidate.Prompt = fallback
	candidate.Reason = "模型返回非标准 JSON，已自动提取文本。"
	return candidate, nil
}

func normalizeCreativeMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "video", "ecommerce":
		return mode
	default:
		return "image"
	}
}

func normalizePromptStyle(style string) string {
	style = strings.ToLower(strings.TrimSpace(style))
	switch style {
	case "creative", "commercial", "detail":
		return style
	default:
		return "balanced"
	}
}

func normalizeFallbackPrompt(content string) string {
	s := strings.TrimSpace(stripCodeFence(content))
	if s == "" || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return ""
	}
	return s
}

func stripCodeFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func extractFirstJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func buildDeepSeekEndpoint(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}
	return baseURL + "/v1/chat/completions"
}
