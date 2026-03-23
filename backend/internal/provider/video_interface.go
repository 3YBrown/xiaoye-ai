package provider

// VideoProvider 视频生成服务商接口
type VideoProvider interface {
	// GetProviderName 返回服务商名称
	GetProviderName() string

	// IsAvailable 检查服务是否可用
	IsAvailable() bool

	// GetSupportedModels 返回该服务商支持的模型列表
	GetSupportedModels() []VideoModel

	// CreateVideoTask 创建视频生成任务
	CreateVideoTask(req VideoGenerateRequest) (*VideoTaskResult, error)

	// GetVideoTaskStatus 查询视频任务状态
	GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error)

	// CalculateCredits 计算所需钻石
	CalculateCredits(resolution string, duration int, generateAudio bool) int
}

// VideoModel 视频模型信息
type VideoModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
}

// VideoGenerateRequest 视频生成请求（通用）
type VideoGenerateRequest struct {
	Model           string   `json:"model"`            // 模型ID
	Prompt          string   `json:"prompt"`           // 提示词
	Mode            string   `json:"mode"`             // text-to-video, first-frame, first-last-frame
	Resolution      string   `json:"resolution"`       // 480p, 720p, 1080p, 4k
	Ratio           string   `json:"ratio"`            // 16:9, 9:16, 1:1, 4:3, 3:4, 21:9
	Duration        int      `json:"duration"`         // 时长秒
	GenerateAudio   bool     `json:"generate_audio"`   // 是否生成配音
	FirstFrame      string   `json:"first_frame"`      // 首帧图片 base64
	LastFrame       string   `json:"last_frame"`       // 尾帧图片 base64
	ReferenceImages []string `json:"reference_images"` // 参考图 base64 (Veo 3.1, 最多3张)
}

// VideoTaskResult 创建任务结果
type VideoTaskResult struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	VideoURL string `json:"video_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// VideoTaskStatusResponse 任务状态响应（通用）
type VideoTaskStatusResponse struct {
	TaskID       string                 `json:"task_id"`
	Status       string                 `json:"status"` // queued, running, succeeded, failed, expired
	VideoURL     string                 `json:"video_url,omitempty"`
	CoverURL     string                 `json:"cover_url,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	RawResponse  map[string]interface{} `json:"raw_response,omitempty"` // 原始响应
}

// 模型到服务商的映射
var modelProviderMap = map[string]string{
	// 火山引擎 - Seedance-1.5
	"doubao-seedance-1-5-pro-251215": "volcengine",
	// Google Veo 3.1
	"veo-3.1-generate-preview": "google",
	// 快手可灵 (未来)
	"kling-v1": "kling",
}

// GetProviderByModel 根据模型ID获取服务商名称
func GetProviderByModel(modelID string) string {
	if provider, ok := modelProviderMap[modelID]; ok {
		return provider
	}
	return ""
}

// RegisterModel 注册新模型（用于动态扩展）
func RegisterModel(modelID, providerName string) {
	modelProviderMap[modelID] = providerName
}

// videoProviderRegistry 服务商注册表
var videoProviderRegistry = make(map[string]VideoProvider)

// RegisterVideoProvider 注册视频服务商
func RegisterVideoProvider(name string, provider VideoProvider) {
	videoProviderRegistry[name] = provider
}

// GetVideoProvider 获取视频服务商
func GetVideoProvider(name string) VideoProvider {
	return videoProviderRegistry[name]
}

// GetVideoProviderForModel 根据模型ID获取对应的服务商
func GetVideoProviderForModel(modelID string) VideoProvider {
	providerName := GetProviderByModel(modelID)
	if providerName == "" {
		return nil
	}
	return GetVideoProvider(providerName)
}

// GetAllVideoModels 获取所有可用模型
func GetAllVideoModels() []VideoModel {
	var models []VideoModel
	for _, provider := range videoProviderRegistry {
		if provider.IsAvailable() {
			models = append(models, provider.GetSupportedModels()...)
		}
	}
	return models
}
