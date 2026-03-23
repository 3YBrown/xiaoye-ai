package provider

import (
	"errors"
	"fmt"
	"net/url"
)

// sanitizeHTTPError 从 HTTP 请求错误中提取根因，去除包含敏感信息（如 API 密钥）的完整 URL
func sanitizeHTTPError(err error) string {
	// 逐层 Unwrap 取最内层错误，Go 的 http 错误链: url.Error -> 具体原因
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err.Error()
	}
	return err.Error()
}

// ImageGenerator 图像生成器通用接口
type ImageGenerator interface {
	// GenerateImage 生成图像
	GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error)
	// Name 返回模型显示名称
	Name() string
	// ID 返回模型唯一标识
	ID() string
	// Provider 返回提供商名称
	Provider() string
	// IsAvailable 检查服务是否可用（API Key 是否配置）
	IsAvailable() bool
}

// MultiImageGenerator 多图生成器接口（电商中心使用）
type MultiImageGenerator interface {
	ImageGenerator
	// GenerateMultiImage 多图生成（多图生组图）
	// inputImages: 输入图片列表（Base64格式）
	// outputCount: 期望输出的图片数量
	GenerateMultiImage(prompt string, inputImages []string, outputCount int, opts ImageOptions) (*MultiImageResult, error)
	// SupportsMultiImage 是否支持多图生成
	SupportsMultiImage() bool
}

// ImageOptions 通用图像生成选项
type ImageOptions struct {
	AspectRatio string   // 宽高比: "1:1", "16:9", "9:16", "4:3", "3:4"
	ImageSize   string   // 图像尺寸: "1K", "2K", "4K"
	InputImages []string // 输入图像的 Base64 数据列表（用于图片编辑）
	MaskImage   string   // Base64 mask 图片（用于局部重绘 inpainting）
}

// ImageResult 单图像生成结果
type ImageResult struct {
	Data     string // Base64 编码的图像数据
	MimeType string // 图像 MIME 类型
}

// MultiImageResult 多图像生成结果
type MultiImageResult struct {
	Images   []ImageResult // 多张图像结果
	MimeType string        // 图像 MIME 类型
}

// ModelInfo 模型信息（用于前端展示）
type ModelInfo struct {
	ID          string `json:"id"`          // 模型唯一标识
	Name        string `json:"name"`        // 模型显示名称
	Provider    string `json:"provider"`    // 提供商: gemini, volcengine
	Description string `json:"description"` // 模型描述
	Available   bool   `json:"available"`   // 是否可用
}

// 已注册的模型
var models = make(map[string]ImageGenerator)

// Register 注册一个图像生成模型
func Register(id string, generator ImageGenerator) {
	models[id] = generator
}

// Get 获取指定的图像生成模型
func Get(id string) (ImageGenerator, error) {
	if g, ok := models[id]; ok {
		if !g.IsAvailable() {
			return nil, fmt.Errorf("模型 %s 未配置或不可用", id)
		}
		return g, nil
	}
	return nil, fmt.Errorf("未知的图像生成模型: %s", id)
}

// GetDefault 获取默认的图像生成模型
func GetDefault() (ImageGenerator, error) {
	// 默认优先级：Nanobanana Pro 优先，Seedream-4.5 备选
	priority := []string{"gemini-3-pro-image-preview", "doubao-seedream-4-5"}
	for _, id := range priority {
		if g, ok := models[id]; ok && g.IsAvailable() {
			return g, nil
		}
	}
	return nil, fmt.Errorf("没有可用的图像生成模型")
}

// modelOrder 定义模型在列表中的显示顺序
var modelOrder = []string{
	"gemini-3.1-flash-image-preview",
	"gemini-3-pro-image-preview",
	"doubao-seedream-4-5",
}

// ListAvailable 列出所有模型信息（按 modelOrder 排序）
func ListAvailable() []ModelInfo {
	var result []ModelInfo
	seen := make(map[string]bool)
	// 按指定顺序添加
	for _, id := range modelOrder {
		if g, ok := models[id]; ok {
			result = append(result, ModelInfo{
				ID:          id,
				Name:        g.Name(),
				Provider:    g.Provider(),
				Description: "",
				Available:   g.IsAvailable(),
			})
			seen[id] = true
		}
	}
	// 追加未在 modelOrder 中的模型
	for id, g := range models {
		if seen[id] {
			continue
		}
		result = append(result, ModelInfo{
			ID:          id,
			Name:        g.Name(),
			Provider:    g.Provider(),
			Description: "",
			Available:   g.IsAvailable(),
		})
	}
	return result
}
