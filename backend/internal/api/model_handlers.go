package api

import (
	"net/http"

	"google-ai-proxy/internal/provider"

	"github.com/gin-gonic/gin"
)

// GetModels 获取所有可用的图像生成模型列表
// GET /api/models
func GetModels(c *gin.Context) {
	models := provider.ListAvailable()

	// 补充模型显示名称（优先使用 ModelDisplayNames 中定义的友好名称）
	type ModelResponse struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Provider    string `json:"provider"`
		Description string `json:"description"`
		Available   bool   `json:"available"`
	}

	result := make([]ModelResponse, 0, len(models))
	for _, m := range models {
		// 使用 GetModelDisplayName 获取友好名称，如果不存在则使用模型自带名称
		displayName := GetModelDisplayName(m.ID)
		if displayName == m.ID {
			displayName = m.Name
		}

		result = append(result, ModelResponse{
			ID:          m.ID,
			Name:        displayName,
			Provider:    m.Provider,
			Description: m.Description,
			Available:   m.Available,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": result,
	})
}
