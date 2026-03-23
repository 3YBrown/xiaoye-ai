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

const (
	CreditTxTypeReversePromptCost = "reverse_prompt_cost"
	reversePromptCredits          = 2
	reversePromptModel            = "doubao-seed-2-0-pro-260215"
)

// ReversePromptRequest 图片反推提示词请求。
type ReversePromptRequest struct {
	Image       string `json:"image" binding:"required"`
	Language    string `json:"language,omitempty"`
	TargetModel string `json:"target_model,omitempty"`
}

// ReversePromptResponse 图片反推提示词响应。
type ReversePromptResponse struct {
	Prompt string                 `json:"prompt"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

func ReversePrompt(c *gin.Context) {
	startTime := time.Now()
	userID := c.GetUint64("userID")
	userIDStr := strconv.FormatUint(userID, 10)

	var req ReversePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := gin.H{"error": "invalid request"}
		logAPICall("/api/tools/reverse-prompt", nil, http.StatusBadRequest, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		resp := gin.H{"error": "image is required"}
		logAPICall("/api/tools/reverse-prompt", nil, http.StatusBadRequest, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	req.Language = normalizeReversePromptLanguage(req.Language)
	req.TargetModel = strings.TrimSpace(req.TargetModel)
	if req.TargetModel == "" {
		req.TargetModel = "Nanobanana Pro"
	}

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	if err := deductReversePromptCredits(userID, user.Credits, reversePromptCredits); err != nil {
		resp := gin.H{
			"error":            err.Error(),
			"required_credits": reversePromptCredits,
			"current_balance":  user.Credits,
		}
		logAPICall("/api/tools/reverse-prompt", nil, http.StatusPaymentRequired, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusPaymentRequired, resp)
		return
	}

	prompt, usage, err := callReversePromptAPI(req)
	if err != nil {
		refundCredits(userID, reversePromptCredits, "reverse-prompt-request-failed")
		log.Printf("[ReversePrompt] failed [user:%d]: %v", userID, err)
		resp := gin.H{"error": "图片反推失败，请重试"}
		logAPICall("/api/tools/reverse-prompt", nil, http.StatusBadGateway, resp, time.Since(startTime), userIDStr)
		c.JSON(http.StatusBadGateway, resp)
		return
	}

	creditsRemaining := user.Credits - reversePromptCredits
	if creditsRemaining < 0 {
		creditsRemaining = 0
	}

	resp := ReversePromptResponse{
		Prompt: prompt,
		Meta: map[string]interface{}{
			"provider":          "volcengine",
			"model":             reversePromptModel,
			"language":          req.Language,
			"target_model":      req.TargetModel,
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
			"credits_spent":     reversePromptCredits,
			"credits_remaining": creditsRemaining,
			"latency_ms":        time.Since(startTime).Milliseconds(),
		},
	}
	logAPICall("/api/tools/reverse-prompt", nil, http.StatusOK, resp, time.Since(startTime), userIDStr)
	c.JSON(http.StatusOK, resp)
}

func deductReversePromptCredits(userID uint64, currentCredits, requiredCredits int) error {
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
		CreditTxTypeReversePromptCost,
		"reverse_prompt",
		"",
		"图片反推提示词",
	); err != nil {
		refundCredits(userID, requiredCredits, "reverse-prompt-ledger-write-failed")
		return errors.New("记录钻石流水失败")
	}
	return nil
}

type volcengineChatRequest struct {
	Model    string                  `json:"model"`
	Messages []volcengineChatMessage `json:"messages"`
}

type volcengineChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type volcengineChatContentPart struct {
	Type     string                  `json:"type"`
	Text     string                  `json:"text,omitempty"`
	ImageURL *volcengineChatImageURL `json:"image_url,omitempty"`
}

type volcengineChatImageURL struct {
	URL string `json:"url"`
}

type volcengineChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func callReversePromptAPI(req ReversePromptRequest) (string, struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}, error) {
	var emptyUsage struct {
		PromptTokens     int
		CompletionTokens int
		TotalTokens      int
	}

	apiKey := strings.TrimSpace(config.GetVolcengineAPIKey())
	if apiKey == "" {
		return "", emptyUsage, errors.New("ARK_API_KEY is not configured")
	}

	systemPrompt := buildReversePromptSystemPrompt(req.Language, req.TargetModel)

	imageURL := req.Image
	if !strings.HasPrefix(imageURL, "data:") && !strings.HasPrefix(imageURL, "http") {
		imageURL = "data:image/jpeg;base64," + imageURL
	}

	userContent := []volcengineChatContentPart{
		{
			Type:     "image_url",
			ImageURL: &volcengineChatImageURL{URL: imageURL},
		},
		{
			Type: "text",
			Text: "请分析这张图片，反推出可以生成该图片的提示词。",
		},
	}

	chatReq := volcengineChatRequest{
		Model: reversePromptModel,
		Messages: []volcengineChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	bodyBytes, err := json.Marshal(chatReq)
	if err != nil {
		return "", emptyUsage, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", emptyUsage, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return "", emptyUsage, fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", emptyUsage, fmt.Errorf("read response: %w", err)
	}
	if httpResp.StatusCode >= http.StatusBadRequest {
		return "", emptyUsage, fmt.Errorf("volcengine status=%d body=%s", httpResp.StatusCode, string(respBytes))
	}

	var parsed volcengineChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", emptyUsage, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", emptyUsage, errors.New("volcengine returned empty choices")
	}

	prompt := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if prompt == "" {
		return "", emptyUsage, errors.New("volcengine returned empty content")
	}

	emptyUsage.PromptTokens = parsed.Usage.PromptTokens
	emptyUsage.CompletionTokens = parsed.Usage.CompletionTokens
	emptyUsage.TotalTokens = parsed.Usage.TotalTokens

	return prompt, emptyUsage, nil
}

func buildReversePromptSystemPrompt(language, targetModel string) string {
	langInstruction := "请使用中文输出提示词。"
	if language == "en" {
		langInstruction = "Please output the prompt in English."
	}

	lines := []string{
		"你是一位专业的 AI 图像提示词反推专家。",
		"你的任务是：根据用户提供的图片，反推出能够尽可能还原该图片的 AI 绘画提示词。",
		"",
		"输出要求：",
		"1. 仅输出提示词文本，不要输出任何解释、标题、前缀或额外说明。",
		"2. 提示词应包含：主体描述、场景/背景、构图/视角、光影/色彩、艺术风格/质感。",
		"3. 按重要性排列，核心主体和动作在前，氛围和细节在后。",
		"4. 使用清晰、可执行的描述，避免模糊词汇。",
		"5. 提示词长度适中，既要覆盖画面关键元素，又不过度冗长。",
		"",
		langInstruction,
		fmt.Sprintf("目标生成模型为 %s，请按照该模型的提示词风格和最佳实践来输出。", targetModel),
	}

	return strings.Join(lines, "\n")
}

func normalizeReversePromptLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "en", "english":
		return "en"
	default:
		return "zh"
	}
}
