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

// VolcengineModel 火山引擎 Seedream-4.5 图像生成模型
type VolcengineModel struct {
	id               string // 模型 ID
	name             string // 显示名称
	endpoint         string // API 中使用的模型名称
	supportsMulti    bool   // 是否支持多图生组图
	defaultMaxImages int    // 默认最大输出图片数
}

func init() {
	// 注册火山引擎的图像生成模型

	// Seedream-4.5 - 最新版，支持多图生组图
	Register("doubao-seedream-4-5", &VolcengineModel{
		id:               "doubao-seedream-4-5",
		name:             "Seedream-4.5",
		endpoint:         "doubao-seedream-4-5-251128",
		supportsMulti:    true,
		defaultMaxImages: 7,
	})
}

func (v *VolcengineModel) ID() string {
	return v.id
}

func (v *VolcengineModel) Name() string {
	return v.name
}

func (v *VolcengineModel) Provider() string {
	return "volcengine"
}

func (v *VolcengineModel) IsAvailable() bool {
	return config.GetVolcengineAPIKey() != ""
}

func (v *VolcengineModel) SupportsMultiImage() bool {
	return v.supportsMulti
}

// GenerateImage 单图生成
func (v *VolcengineModel) GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error) {
	apiKey := config.GetVolcengineAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	apiURL := "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	size := v.convertSize(opts.ImageSize, opts.AspectRatio)

	// 将宽高比描述追加到 prompt 中（方式1要求）
	fullPrompt := prompt + v.getAspectRatioHint(opts.AspectRatio)

	reqBody := volcengineImageRequest{
		Model:          v.endpoint,
		Prompt:         fullPrompt,
		Size:           size,
		ResponseFormat: "b64_json",
		Watermark:      false,
	}

	// 如果是支持多图的模型，需要关闭组图功能生成单图
	if v.supportsMulti {
		reqBody.SequentialImageGeneration = "disabled"
	}

	// 支持图片输入（图生图）- 使用 Image 数组
	if len(opts.InputImages) > 0 {
		images := make([]string, 0, len(opts.InputImages))
		for _, img := range opts.InputImages {
			images = append(images, "data:image/png;base64,"+img)
		}
		reqBody.Image = images
	}

	return v.doRequest(apiURL, apiKey, reqBody)
}

// GenerateMultiImage 多图生组图（电商中心功能）
func (v *VolcengineModel) GenerateMultiImage(prompt string, inputImages []string, outputCount int, opts ImageOptions) (*MultiImageResult, error) {
	if !v.supportsMulti {
		return nil, fmt.Errorf("模型 %s 不支持多图生组图功能", v.name)
	}

	apiKey := config.GetVolcengineAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("服务未配置，请联系管理员")
	}

	// 验证输入图片数量
	if len(inputImages) < 1 || len(inputImages) > 14 {
		return nil, fmt.Errorf("多图生组图需要1-14张输入图片，当前: %d张", len(inputImages))
	}

	// 计算最大输出数量 (输入图数 + 输出图数 <= 15)
	maxOutput := 15 - len(inputImages)
	if outputCount <= 0 || outputCount > maxOutput {
		outputCount = maxOutput
		if outputCount > v.defaultMaxImages {
			outputCount = v.defaultMaxImages
		}
	}

	apiURL := "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	size := v.convertSize(opts.ImageSize, opts.AspectRatio)

	// 将宽高比描述追加到 prompt 中（方式1要求）
	fullPrompt := prompt + v.getAspectRatioHint(opts.AspectRatio)

	// 构建输入图片列表（base64 格式）
	images := make([]string, len(inputImages))
	for i, img := range inputImages {
		images[i] = "data:image/png;base64," + img
	}

	reqBody := volcengineImageRequest{
		Model:                     v.endpoint,
		Prompt:                    fullPrompt,
		Size:                      size,
		ResponseFormat:            "b64_json",
		Watermark:                 false,
		SequentialImageGeneration: "auto", // 启用组图功能
		SequentialImageGenOptions: &volcengineSeqOptions{MaxImages: outputCount},
		Image:                     images,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[Volcengine] 错误: 请求体序列化失败 - %v", err)
		return nil, err
	}

	client := &http.Client{Timeout: 600 * time.Second} // 多图生成需要更长超时
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[Volcengine] 正在调用 Seedream-4.5 API 生成组图，输入 %d 张，期望输出 %d 张...", len(inputImages), outputCount)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Volcengine] 错误: HTTP 请求失败 - %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("[Volcengine] 错误: API 返回错误状态码 %d, 响应: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("%s", parseVolcengineError(body))
	}

	var genResp volcengineImageResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		log.Printf("[Volcengine] 错误: 响应反序列化失败 - %v", err)
		return nil, fmt.Errorf("API 响应解析失败: %v", err)
	}

	if len(genResp.Data) == 0 {
		return nil, fmt.Errorf("生成失败: 响应中没有图像数据")
	}

	// 构建多图结果
	results := make([]ImageResult, 0, len(genResp.Data))
	for _, imgData := range genResp.Data {
		if imgData.B64JSON != "" {
			results = append(results, ImageResult{
				Data:     imgData.B64JSON,
				MimeType: "image/png",
			})
		} else if imgData.Error != nil {
			log.Printf("[Volcengine] 警告: 组图中某张图片生成失败 - %s: %s", imgData.Error.Code, imgData.Error.Message)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("所有图片生成均失败")
	}

	log.Printf("[Volcengine] 组图生成成功，共 %d 张图片", len(results))
	return &MultiImageResult{
		Images:   results,
		MimeType: "image/png",
	}, nil
}

func (v *VolcengineModel) doRequest(apiURL, apiKey string, reqBody volcengineImageRequest) (*ImageResult, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[Volcengine] 错误: 请求体序列化失败 - %v", err)
		return nil, err
	}

	client := &http.Client{Timeout: 300 * time.Second}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[Volcengine] 正在调用火山 API 生成图像...")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Volcengine] 错误: HTTP 请求失败 - %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("[Volcengine] 错误: API 返回错误状态码 %d, 响应: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("%s", parseVolcengineError(body))
	}

	var genResp volcengineImageResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		log.Printf("[Volcengine] 错误: 响应反序列化失败 - %v", err)
		return nil, fmt.Errorf("API 响应解析失败: %v", err)
	}

	if len(genResp.Data) == 0 {
		return nil, fmt.Errorf("生成失败: 响应中没有图像数据")
	}

	log.Printf("[Volcengine] 图像生成成功")
	return &ImageResult{
		Data:     genResp.Data[0].B64JSON,
		MimeType: "image/png",
	}, nil
}

func (v *VolcengineModel) convertSize(imageSize, aspectRatio string) string {
	// 根据官方文档方式1：直接传 2K 或 4K，由模型根据 prompt 自动判断宽高比
	if imageSize == "4K" {
		return "4K"
	}
	return "2K" // 默认 2K
}

// getAspectRatioHint 将宽高比转换为自然语言描述，追加到 prompt 中
func (v *VolcengineModel) getAspectRatioHint(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "，横版16:9宽屏比例"
	case "9:16":
		return "，竖版9:16手机屏幕比例"
	case "4:3":
		return "，横版4:3比例"
	case "3:4":
		return "，竖版3:4比例"
	case "3:2":
		return "，横版3:2比例"
	case "2:3":
		return "，竖版2:3比例"
	case "21:9":
		return "，超宽21:9电影比例"
	case "4:5":
		return "，竖版4:5比例"
	case "5:4":
		return "，横版5:4比例"
	default: // 1:1
		return "，正方形1:1比例"
	}
}

// ============ 火山引擎专用结构体 ============

type volcengineImageRequest struct {
	Model                     string                `json:"model"`
	Prompt                    string                `json:"prompt"`
	Size                      string                `json:"size,omitempty"`
	ResponseFormat            string                `json:"response_format,omitempty"`
	Watermark                 bool                  `json:"watermark"`
	Image                     []string              `json:"image,omitempty"`                       // 图片 URL 或 base64 字符串数组
	SequentialImageGeneration string                `json:"sequential_image_generation,omitempty"` // auto/disabled
	SequentialImageGenOptions *volcengineSeqOptions `json:"sequential_image_generation_options,omitempty"`
}

type volcengineSeqOptions struct {
	MaxImages int `json:"max_images"`
}

type volcengineImageResponse struct {
	Model   string                `json:"model"`
	Created int64                 `json:"created"`
	Data    []volcengineImageData `json:"data"`
	Usage   *volcengineUsage      `json:"usage,omitempty"`
	Error   *volcengineError      `json:"error,omitempty"`
}

type volcengineImageData struct {
	B64JSON string           `json:"b64_json,omitempty"`
	URL     string           `json:"url,omitempty"`
	Size    string           `json:"size,omitempty"`
	Error   *volcengineError `json:"error,omitempty"`
}

type volcengineUsage struct {
	GeneratedImages int `json:"generated_images"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

type volcengineError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// volcengineErrorResponse 用于解析 API 错误响应
type volcengineErrorResponse struct {
	Error *volcengineError `json:"error"`
}

// parseVolcengineError 从错误响应中提取错误信息
func parseVolcengineError(body []byte) string {
	var errResp volcengineErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil && errResp.Error.Message != "" {
		return errResp.Error.Message
	}
	// 如果解析失败，返回原始内容
	return string(body)
}
