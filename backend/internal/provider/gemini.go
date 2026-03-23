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
	"time"

	"google-ai-proxy/internal/config"
)

// GeminiModel Gemini 图像生成模型
type GeminiModel struct {
	id       string // 模型 ID
	name     string // 显示名称
	endpoint string // API 端点中的模型名称
}

func init() {
	// 注册 Gemini 的图像生成模型
	Register("gemini-3-pro-image-preview", &GeminiModel{
		id:       "gemini-3-pro-image-preview",
		name:     "Nanobanana Pro",
		endpoint: "gemini-3-pro-image-preview",
	})
	Register("gemini-3.1-flash-image-preview", &GeminiModel{
		id:       "gemini-3.1-flash-image-preview",
		name:     "Nanobanana 2",
		endpoint: "gemini-3.1-flash-image-preview",
	})
}

func (g *GeminiModel) ID() string {
	return g.id
}

func (g *GeminiModel) Name() string {
	return g.name
}

func (g *GeminiModel) Provider() string {
	return "gemini"
}

func (g *GeminiModel) IsAvailable() bool {
	return config.GetGoogleAPIKey() != ""
}

func (g *GeminiModel) GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error) {
	apiKey := config.GetGoogleAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	// 使用模型自己的 endpoint
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.endpoint, apiKey)

	genConfig := &geminiGenerationConfig{
		ResponseModalities: []string{"IMAGE"},
	}

	imageConfig := &geminiImageConfig{}
	if opts.AspectRatio != "" {
		imageConfig.AspectRatio = opts.AspectRatio
	} else {
		imageConfig.AspectRatio = "1:1"
	}
	if opts.ImageSize != "" {
		// Gemini API 使用 "512px" 而非 "0.5K"
		if opts.ImageSize == "0.5K" {
			imageConfig.ImageSize = "512px"
		} else {
			imageConfig.ImageSize = opts.ImageSize
		}
	} else {
		imageConfig.ImageSize = "1K"
	}
	genConfig.ImageConfig = imageConfig

	parts := []geminiPart{
		{Text: prompt},
	}

	// 添加输入图片（统一使用 InputImages 数组）
	for _, img := range opts.InputImages {
		parts = append(parts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: "image/png",
				Data:     img,
			},
		})
	}

	// 添加 mask 图片（局部重绘 inpainting）
	if opts.MaskImage != "" {
		parts = append(parts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: "image/png",
				Data:     opts.MaskImage,
			},
		})
	}

	reqBody := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Parts: parts,
			},
		},
		GenerationConfig: genConfig,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[Gemini] 错误: 请求体序列化失败 - %v", err)
		return nil, err
	}

	client := g.createHTTPClient()
	log.Printf("[Gemini] 正在调用 API 生成图像...")

	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Gemini] 错误: HTTP 请求失败 - %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("[Gemini] 错误: API 返回错误状态码 %d", resp.StatusCode)
		return nil, fmt.Errorf("生成失败 (状态码 %d)", resp.StatusCode)
	}

	var genResp geminiGenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		log.Printf("[Gemini] 错误: 响应反序列化失败 - %v", err)
		return nil, fmt.Errorf("API 响应解析失败: %v", err)
	}

	if len(genResp.Candidates) == 0 {
		log.Printf("[Gemini] 错误: API 响应中没有候选结果")
		return nil, fmt.Errorf("响应中没有候选结果")
	}

	candidate := genResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		log.Printf("[Gemini] 错误: 候选结果中没有内容部分")
		return nil, fmt.Errorf("候选结果中没有内容部分")
	}

	var lastImageData string
	for _, part := range candidate.Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			lastImageData = part.InlineData.Data
		}
	}

	if lastImageData == "" {
		log.Printf("[Gemini] 错误: 响应中未找到有效的图像数据")
		return nil, fmt.Errorf("响应部分中未找到图像数据")
	}

	log.Printf("[Gemini] 图像生成成功，获得 %d 字节的图像数据", len(lastImageData))
	return &ImageResult{
		Data:     lastImageData,
		MimeType: "image/png",
	}, nil
}

func (g *GeminiModel) createHTTPClient() *http.Client {
	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		return &http.Client{
			Timeout: 300 * time.Second,
		}
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		log.Printf("[Gemini] 代理地址解析失败: %v, 不使用代理", err)
		return &http.Client{
			Timeout: 300 * time.Second,
		}
	}

	log.Printf("[Gemini] 代理已启用 (%s)", proxy)
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 300 * time.Second,
	}
}

// ============ Gemini 专用结构体 ============

type geminiGenerateRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiGenerationConfig struct {
	ResponseModalities []string           `json:"responseModalities,omitempty"`
	ImageConfig        *geminiImageConfig `json:"imageConfig,omitempty"`
}

type geminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenerateResponse struct {
	Candidates    []geminiCandidate      `json:"candidates"`
	UsageMetadata map[string]interface{} `json:"usageMetadata,omitempty"`
	ModelVersion  string                 `json:"modelVersion,omitempty"`
	ResponseID    string                 `json:"responseId,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
	Index        int           `json:"index,omitempty"`
}
