package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"google-ai-proxy/internal/config"
)

// VolcengineVideoProvider 火山引擎视频生成 Seedance-1.5
type VolcengineVideoProvider struct{}

// 火山引擎支持的模型
var volcengineVideoModels = []VideoModel{
	{
		ID:          "doubao-seedance-1-5-pro-251215",
		Name:        "Seedance-1.5",
		Provider:    "volcengine",
		Description: "火山引擎视频生成",
	},
}

const (
	volcengineVideoAPIEndpoint = "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks"
)

// volcengineVideoRequest 火山引擎视频生成API请求体
type volcengineVideoRequest struct {
	Model         string                   `json:"model"`
	Content       []volcengineVideoContent `json:"content"`
	Resolution    string                   `json:"resolution,omitempty"`
	Ratio         string                   `json:"ratio,omitempty"`
	Duration      int                      `json:"duration,omitempty"`
	GenerateAudio bool                     `json:"generate_audio"`
	Watermark     bool                     `json:"watermark"`
}

type volcengineVideoContent struct {
	Type     string                   `json:"type"`                // text, image_url
	Text     string                   `json:"text,omitempty"`      // 文本内容
	ImageURL *volcengineVideoImageURL `json:"image_url,omitempty"` // 图片内容
	Role     string                   `json:"role,omitempty"`      // first_frame, last_frame
}

type volcengineVideoImageURL struct {
	URL string `json:"url"` // 图片URL或base64
}

// volcengineVideoResponse 创建任务响应
type volcengineVideoResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// volcengineVideoTaskResponse 查询任务响应
type volcengineVideoTaskResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"` // queued, running, succeeded, failed, expired
	Content *struct {
		VideoURL     string `json:"video_url"`
		LastFrameURL string `json:"last_frame_url,omitempty"`
	} `json:"content,omitempty"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Usage *struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
	Seed                  int64  `json:"seed"`
	Resolution            string `json:"resolution"`
	Ratio                 string `json:"ratio"`
	Duration              int    `json:"duration"`
	FramesPerSecond       int    `json:"framespersecond"`
	GenerateAudio         bool   `json:"generate_audio"`
	Draft                 bool   `json:"draft"`
	ServiceTier           string `json:"service_tier"`
	ExecutionExpiresAfter int    `json:"execution_expires_after"`
}

// NewVolcengineVideoProvider 创建火山引擎视频Provider
func NewVolcengineVideoProvider() *VolcengineVideoProvider {
	return &VolcengineVideoProvider{}
}

// GetProviderName 返回服务商名称
func (v *VolcengineVideoProvider) GetProviderName() string {
	return "volcengine"
}

// IsAvailable 检查服务是否可用
func (v *VolcengineVideoProvider) IsAvailable() bool {
	return config.GetVolcengineAPIKey() != ""
}

// GetSupportedModels 返回支持的模型列表
func (v *VolcengineVideoProvider) GetSupportedModels() []VideoModel {
	return volcengineVideoModels
}

// CalculateCredits 计算视频生成所需钻石
func (v *VolcengineVideoProvider) CalculateCredits(resolution string, duration int, generateAudio bool) int {
	// 每秒基础价格
	basePerSecond := map[string]float64{
		"480p":  6.0,
		"720p":  10.0,
		"1080p": 16.0,
	}

	base, ok := basePerSecond[resolution]
	if !ok {
		base = 10.0
	}

	credits := base * float64(duration)
	if generateAudio {
		credits *= 1.2
	}
	return int(credits)
}

// CreateVideoTask 创建视频生成任务
func (v *VolcengineVideoProvider) CreateVideoTask(req VideoGenerateRequest) (*VideoTaskResult, error) {
	apiKey := config.GetVolcengineAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	// 构建请求内容
	content := make([]volcengineVideoContent, 0)

	// 添加文本内容
	if req.Prompt != "" {
		content = append(content, volcengineVideoContent{
			Type: "text",
			Text: req.Prompt,
		})
	}

	// 根据模式添加图片
	switch req.Mode {
	case "first-frame":
		if req.FirstFrame != "" {
			content = append(content, volcengineVideoContent{
				Type: "image_url",
				ImageURL: &volcengineVideoImageURL{
					URL: "data:image/png;base64," + req.FirstFrame,
				},
				Role: "first_frame",
			})
		}
	case "first-last-frame":
		if req.FirstFrame != "" {
			content = append(content, volcengineVideoContent{
				Type: "image_url",
				ImageURL: &volcengineVideoImageURL{
					URL: "data:image/png;base64," + req.FirstFrame,
				},
				Role: "first_frame",
			})
		}
		if req.LastFrame != "" {
			content = append(content, volcengineVideoContent{
				Type: "image_url",
				ImageURL: &volcengineVideoImageURL{
					URL: "data:image/png;base64," + req.LastFrame,
				},
				Role: "last_frame",
			})
		}
	}

	// 确定使用的模型（火山引擎只支持自己的模型）
	modelID := req.Model
	if modelID == "" {
		modelID = volcengineVideoModels[0].ID
	}

	// 构建请求体
	reqBody := volcengineVideoRequest{
		Model:         modelID,
		Content:       content,
		Resolution:    req.Resolution,
		Ratio:         req.Ratio,
		Duration:      req.Duration,
		GenerateAudio: req.GenerateAudio,
		Watermark:     false,
	}

	// 设置默认值
	if reqBody.Resolution == "" {
		reqBody.Resolution = "720p"
	}
	if reqBody.Ratio == "" {
		reqBody.Ratio = "16:9"
	}
	if reqBody.Duration == 0 || reqBody.Duration < 4 {
		reqBody.Duration = 5
	}
	if reqBody.Duration > 12 {
		reqBody.Duration = 12
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[VolcengineVideo] 错误: 请求体序列化失败 - %v", err)
		return nil, err
	}

	log.Printf("[VolcengineVideo] 创建视频任务: mode=%s, resolution=%s, ratio=%s, duration=%d, audio=%v",
		req.Mode, reqBody.Resolution, reqBody.Ratio, reqBody.Duration, reqBody.GenerateAudio)

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second}
	httpReq, err := http.NewRequest("POST", volcengineVideoAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[VolcengineVideo] 请求失败: %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[VolcengineVideo] API返回错误: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("生成失败 (状态码 %d)", resp.StatusCode)
	}

	var result volcengineVideoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[VolcengineVideo] 解析响应失败: %v, body=%s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Error != nil {
		log.Printf("[VolcengineVideo] 任务创建失败: code=%s, message=%s", result.Error.Code, result.Error.Message)
		return nil, fmt.Errorf("创建任务失败: %s", result.Error.Message)
	}

	log.Printf("[VolcengineVideo] 任务创建成功: task_id=%s", result.ID)

	return &VideoTaskResult{
		TaskID: result.ID,
		Status: "queued",
	}, nil
}

// GetVideoTaskStatus 查询视频任务状态
func (v *VolcengineVideoProvider) GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error) {
	apiKey := config.GetVolcengineAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	url := volcengineVideoAPIEndpoint + "/" + taskID

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[VolcengineVideo] 查询任务失败: %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[VolcengineVideo] 查询任务返回错误: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("查询任务返回错误: %s", string(body))
	}

	var result volcengineVideoTaskResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[VolcengineVideo] 解析任务状态失败: %v", err)
		return nil, fmt.Errorf("解析任务状态失败: %w", err)
	}

	log.Printf("[VolcengineVideo] 任务状态: task_id=%s, status=%s", taskID, result.Status)

	// 转换为通用响应格式
	response := &VideoTaskStatusResponse{
		TaskID: result.ID,
		Status: result.Status,
	}

	if result.Content != nil {
		response.VideoURL = result.Content.VideoURL
	}

	if result.Error != nil {
		response.ErrorCode = result.Error.Code
		response.ErrorMessage = result.Error.Message
	}

	// 保存原始响应
	rawResp := make(map[string]interface{})
	json.Unmarshal(body, &rawResp)
	response.RawResponse = rawResp

	return response, nil
}

// init 注册火山引擎视频Provider
func init() {
	RegisterVideoProvider("volcengine", NewVolcengineVideoProvider())
}
