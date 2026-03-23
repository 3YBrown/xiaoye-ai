package api

import (
	"strconv"
)

// GenerateEcommercePromptSuffix 根据电商平台和图片类型生成提示词后缀
// 生成自然语言格式，例如：生成5张淘宝平台详情图
func GenerateEcommercePromptSuffix(imageType, ecommerceType string, outputCount int) string {
	// 构建自然语言提示词
	suffix := "。生成"

	// 添加张数
	if outputCount > 1 {
		suffix += strconv.Itoa(outputCount) + "张"
	}

	// 添加电商平台
	if ecommerceType != "" {
		suffix += ecommerceType + "平台"
	}

	// 添加图片类型
	if imageType != "" {
		suffix += imageType
	}

	// 如果只有"。生成"说明没有任何有效内容
	if suffix == "。生成" {
		return ""
	}

	return suffix
}
