package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"google-ai-proxy/internal/config"
)

// GoogleVideoProvider Google Veo 视频生成
type GoogleVideoProvider struct{}

const (
	googleVideoAPIBase = "https://generativelanguage.googleapis.com/v1beta"
)

// Google 支持的视频模型
var googleVideoModels = []VideoModel{
	{
		ID:          "veo-3.1-generate-preview",
		Name:        "Veo 3.1",
		Provider:    "google",
		Description: "Google Veo 3.1 视频生成",
	},
}

// Veo 3.1 分辨率阶梯定价 (credits/s)
// 成本基准: $0.40/s ≈ 30 credits/s，按分辨率加价覆盖存储/带宽/利润
var VeoCreditsPerSecond = map[string]int{
	"720p":  45,
	"1080p": 65,
	"4k":    90,
}

// ======== Veo API 请求/响应结构体 ========

type veoInstance struct {
	Prompt          string              `json:"prompt,omitempty"`
	Image           *veoInlineData      `json:"image,omitempty"`
	LastFrame       *veoInlineData      `json:"lastFrame,omitempty"`
	ReferenceImages []veoReferenceImage `json:"referenceImages,omitempty"`
}

type veoReferenceImage struct {
	Image         veoInlineData `json:"image"`
	ReferenceType string        `json:"referenceType"`
}

type veoInlineData struct {
	InlineData *veoMediaData `json:"inlineData"`
}

type veoMediaData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type veoParameters struct {
	AspectRatio      string `json:"aspectRatio,omitempty"`
	DurationSeconds  int    `json:"durationSeconds,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	PersonGeneration string `json:"personGeneration,omitempty"`
}

type veoRequest struct {
	Instances  []veoInstance `json:"instances"`
	Parameters veoParameters `json:"parameters"`
}

// veoOperationResponse 创建任务后返回的 Long Running Operation
type veoOperationResponse struct {
	Name     string                 `json:"name"` // operations/xxx - 用作 taskID
	Done     bool                   `json:"done"`
	Error    *veoErrorInfo          `json:"error,omitempty"`
	Response map[string]interface{} `json:"response,omitempty"`
}

type veoErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ======== 接口实现 ========

func NewGoogleVideoProvider() *GoogleVideoProvider {
	return &GoogleVideoProvider{}
}

func (g *GoogleVideoProvider) GetProviderName() string {
	return "google"
}

func (g *GoogleVideoProvider) IsAvailable() bool {
	return config.GetGoogleAPIKey() != ""
}

func (g *GoogleVideoProvider) GetSupportedModels() []VideoModel {
	return googleVideoModels
}

// CalculateCredits 计算视频生成所需钻石
// Veo 3.1 按分辨率阶梯定价，原生含音频不额外收费
func (g *GoogleVideoProvider) CalculateCredits(resolution string, duration int, generateAudio bool) int {
	base, ok := VeoCreditsPerSecond[resolution]
	if !ok {
		base = VeoCreditsPerSecond["720p"]
	}
	return base * duration
}

// CreateVideoTask 创建 Veo 视频生成任务
func (g *GoogleVideoProvider) CreateVideoTask(req VideoGenerateRequest) (*VideoTaskResult, error) {
	apiKey := config.GetGoogleAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	modelID := req.Model
	if modelID == "" {
		modelID = googleVideoModels[0].ID
	}

	// 构建 instance
	instance := veoInstance{
		Prompt: req.Prompt,
	}

	// 首帧图片
	if req.Mode == "first-frame" || req.Mode == "first-last-frame" {
		if req.FirstFrame != "" {
			instance.Image = &veoInlineData{
				InlineData: &veoMediaData{
					MimeType: "image/png",
					Data:     req.FirstFrame,
				},
			}
		}
	}

	// 尾帧图片
	if req.Mode == "first-last-frame" {
		if req.LastFrame != "" {
			instance.LastFrame = &veoInlineData{
				InlineData: &veoMediaData{
					MimeType: "image/png",
					Data:     req.LastFrame,
				},
			}
		}
	}

	// 参考图（Veo 3.1 独有，最多 3 张）
	if len(req.ReferenceImages) > 0 {
		refs := req.ReferenceImages
		if len(refs) > 3 {
			refs = refs[:3]
		}
		for _, imgBase64 := range refs {
			instance.ReferenceImages = append(instance.ReferenceImages, veoReferenceImage{
				Image: veoInlineData{
					InlineData: &veoMediaData{
						MimeType: "image/png",
						Data:     imgBase64,
					},
				},
				ReferenceType: "asset",
			})
		}
	}

	// 参数映射
	ratio := req.Ratio
	if ratio == "" {
		ratio = "16:9"
	}
	// Veo 仅支持 16:9 和 9:16
	if ratio != "16:9" && ratio != "9:16" {
		ratio = "16:9"
	}

	duration := req.Duration
	if duration <= 0 {
		duration = 8
	}
	// Veo 支持 4, 6, 8
	if duration < 4 {
		duration = 4
	}
	if duration == 5 || duration == 7 {
		duration = duration + 1 // 5→6, 7→8
	}
	if duration > 8 {
		duration = 8
	}

	resolution := req.Resolution
	if resolution == "" {
		resolution = "720p"
	}
	// 官方限制: 1080p 和 4k 必须 duration=8
	if resolution == "1080p" || resolution == "4k" {
		duration = 8
	}

	params := veoParameters{
		AspectRatio:      ratio,
		DurationSeconds:  duration,
		Resolution:       resolution,
		PersonGeneration: "allow_all",
	}

	reqBody := veoRequest{
		Instances:  []veoInstance{instance},
		Parameters: params,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[GoogleVideo] 错误: 请求体序列化失败 - %v", err)
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/models/%s:predictLongRunning", googleVideoAPIBase, modelID)

	log.Printf("[GoogleVideo] 创建视频任务: model=%s, mode=%s, resolution=%s, ratio=%s, duration=%d",
		modelID, req.Mode, resolution, ratio, duration)

	client := g.createHTTPClient()
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[GoogleVideo] 请求失败: %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[GoogleVideo] API返回错误: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("生成失败 (状态码 %d)", resp.StatusCode)
	}

	var result veoOperationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[GoogleVideo] 解析响应失败: %v, body=%s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Error != nil {
		log.Printf("[GoogleVideo] 任务创建失败: code=%d, message=%s", result.Error.Code, result.Error.Message)
		return nil, fmt.Errorf("创建任务失败: %s", result.Error.Message)
	}

	if result.Name == "" {
		log.Printf("[GoogleVideo] 响应中没有 operation name, body=%s", string(body))
		return nil, fmt.Errorf("创建任务失败: 未获取到任务ID")
	}

	log.Printf("[GoogleVideo] 任务创建成功: operation=%s", result.Name)

	return &VideoTaskResult{
		TaskID: result.Name, // operations/xxx
		Status: "queued",
	}, nil
}

// GetVideoTaskStatus 查询 Veo 视频任务状态
func (g *GoogleVideoProvider) GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error) {
	apiKey := config.GetGoogleAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	// taskID 就是 operation name，如 "operations/xxx"
	apiURL := fmt.Sprintf("%s/%s", googleVideoAPIBase, taskID)

	client := g.createHTTPClient()
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[GoogleVideo] 查询任务失败: %v", err)
		return nil, fmt.Errorf("查询失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[GoogleVideo] 查询任务返回错误: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("查询任务返回错误: %s", string(body))
	}

	var result veoOperationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[GoogleVideo] 解析任务状态失败: %v", err)
		return nil, fmt.Errorf("解析任务状态失败: %w", err)
	}

	log.Printf("[GoogleVideo] 任务状态: operation=%s, done=%v", taskID, result.Done)

	response := &VideoTaskStatusResponse{
		TaskID: taskID,
	}

	if result.Error != nil {
		response.Status = "failed"
		response.ErrorCode = strconv.Itoa(result.Error.Code)
		response.ErrorMessage = result.Error.Message
		return response, nil
	}

	if !result.Done {
		response.Status = "running"
		return response, nil
	}

	// done=true，解析视频 URL
	response.Status = "succeeded"
	videoURL := extractVeoVideoURL(result.Response)
	if videoURL != "" {
		response.VideoURL = videoURL
	} else {
		log.Printf("[GoogleVideo] 任务完成但未找到视频URL, response=%v", result.Response)
		response.Status = "failed"
		response.ErrorMessage = "任务完成但未获取到视频"
	}

	// 保存原始响应
	rawResp := make(map[string]interface{})
	json.Unmarshal(body, &rawResp)
	response.RawResponse = rawResp

	return response, nil
}

// extractVeoVideoURL 从 Veo 响应中提取视频 URL
// 响应格式: response.generateVideoResponse.generatedSamples[0].video.uri
func extractVeoVideoURL(resp map[string]interface{}) string {
	if resp == nil {
		return ""
	}

	genResp, ok := resp["generateVideoResponse"].(map[string]interface{})
	if !ok {
		return ""
	}

	samples, ok := genResp["generatedSamples"].([]interface{})
	if !ok || len(samples) == 0 {
		return ""
	}

	sample, ok := samples[0].(map[string]interface{})
	if !ok {
		return ""
	}

	video, ok := sample["video"].(map[string]interface{})
	if !ok {
		return ""
	}

	uri, ok := video["uri"].(string)
	if !ok {
		return ""
	}

	return uri
}

// createHTTPClient 创建 HTTP 客户端（支持代理）
func (g *GoogleVideoProvider) createHTTPClient() *http.Client {
	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		return &http.Client{
			Timeout: 300 * time.Second,
		}
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		log.Printf("[GoogleVideo] 代理地址解析失败: %v, 不使用代理", err)
		return &http.Client{
			Timeout: 300 * time.Second,
		}
	}

	log.Printf("[GoogleVideo] 代理已启用 (%s)", proxy)
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 300 * time.Second,
	}
}

// init 注册 Google 视频 Provider
func init() {
	RegisterVideoProvider("google", NewGoogleVideoProvider())
}
