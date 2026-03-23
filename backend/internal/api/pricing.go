package api

import (
	"net/http"

	"google-ai-proxy/internal/provider"

	"github.com/gin-gonic/gin"
)

// Model IDs
const (
	ModelNanobanana  = "gemini-3-pro-image-preview"
	ModelNanobanana2 = "gemini-3.1-flash-image-preview"
	ModelSeedream45  = "doubao-seedream-4-5"
	ModelSeedance15  = "doubao-seedance-1-5-pro-251215"
	ModelVeo31       = "veo-3.1-generate-preview"
)

var ModelDisplayNames = map[string]string{
	ModelNanobanana:  "🍌 Nanobanana Pro",
	ModelNanobanana2: "🍌 Nanobanana 2",
	ModelSeedream45:  "Seedream-4.5",
	ModelSeedance15:  "Seedance-1.5",
	ModelVeo31:       "Veo 3.1",
}

func GetModelDisplayName(model string) string {
	if name, ok := ModelDisplayNames[model]; ok {
		return name
	}
	return model
}

// ImagePricingConfig image generation pricing
// 1 CNY = 10 credits
var ImagePricingConfig = map[string]map[string]int{
	ModelNanobanana: {
		"1K": 10,
		"2K": 12,
		"4K": 20,
	},
	ModelNanobanana2: {
		"0.5K": 3,
		"1K":   6,
		"2K":   8,
		"4K":   12,
	},
	ModelSeedream45: {
		"2K": 6,
		"4K": 10,
	},
}

// VideoPricingConfig video generation pricing.
// Actual deduction is performed by provider.CalculateCredits.
var VideoPricingConfig = struct {
	BasePerSecond   map[string]int
	AudioMultiplier float64
}{
	BasePerSecond: map[string]int{
		"480p":  6,
		"720p":  10,
		"1080p": 16,
	},
	AudioMultiplier: 1.2,
}

// DefaultEcommerceModel ecommerce uses image model pricing
const DefaultEcommerceModel = ModelSeedream45

// GetImageCredits returns image generation credits
func GetImageCredits(model, size string) int {
	modelPricing, ok := ImagePricingConfig[model]
	if !ok {
		return 10
	}

	credits, ok := modelPricing[size]
	if !ok {
		maxCredits := 0
		for _, v := range modelPricing {
			if v > maxCredits {
				maxCredits = v
			}
		}
		if maxCredits > 0 {
			return maxCredits
		}
		return 10
	}
	return credits
}

// GetEcommerceCredits returns ecommerce credits
func GetEcommerceCredits(size string, count int) int {
	creditsPerImage := GetImageCredits(DefaultEcommerceModel, size)
	return creditsPerImage * count
}

// GetPricing returns complete pricing config
func GetPricing(c *gin.Context) {
	pricing := gin.H{
		"image": []gin.H{
			{
				"model":       ModelNanobanana,
				"model_name":  GetModelDisplayName(ModelNanobanana),
				"description": "基于 Gemini 3 Pro",
				"prices": []gin.H{
					{"size": "1K", "credits": ImagePricingConfig[ModelNanobanana]["1K"], "description": "1024x1024"},
					{"size": "2K", "credits": ImagePricingConfig[ModelNanobanana]["2K"], "description": "2048x2048"},
					{"size": "4K", "credits": ImagePricingConfig[ModelNanobanana]["4K"], "description": "4096x4096"},
				},
			},
			{
				"model":       ModelNanobanana2,
				"model_name":  GetModelDisplayName(ModelNanobanana2),
				"description": "基于 Gemini 3.1 Flash",
				"prices": []gin.H{
					{"size": "0.5K", "credits": ImagePricingConfig[ModelNanobanana2]["0.5K"], "description": "512x512"},
					{"size": "1K", "credits": ImagePricingConfig[ModelNanobanana2]["1K"], "description": "1024x1024"},
					{"size": "2K", "credits": ImagePricingConfig[ModelNanobanana2]["2K"], "description": "2048x2048"},
					{"size": "4K", "credits": ImagePricingConfig[ModelNanobanana2]["4K"], "description": "4096x4096"},
				},
			},
			{
				"model":       ModelSeedream45,
				"model_name":  GetModelDisplayName(ModelSeedream45),
				"description": "火山引擎图像模型",
				"prices": []gin.H{
					{"size": "2K", "credits": ImagePricingConfig[ModelSeedream45]["2K"], "description": "2048x2048"},
					{"size": "4K", "credits": ImagePricingConfig[ModelSeedream45]["4K"], "description": "4096x4096"},
				},
			},
		},
		"video": gin.H{
			"base_per_second":     VideoPricingConfig.BasePerSecond,
			"audio_multiplier":    VideoPricingConfig.AudioMultiplier,
			"veo_base_per_second": provider.VeoCreditsPerSecond,
		},
		"ecommerce": gin.H{
			"model":      DefaultEcommerceModel,
			"model_name": GetModelDisplayName(DefaultEcommerceModel),
			"prices":     ImagePricingConfig[DefaultEcommerceModel],
		},
		"exchange_rate": "1=10",
	}

	c.JSON(http.StatusOK, pricing)
}
