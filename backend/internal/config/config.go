package config

import (
	"os"
	"strconv"
)

// GetGoogleAPIKey 获取 Google API Key。
func GetGoogleAPIKey() string {
	return os.Getenv("GOOGLE_API_KEY")
}

// GetDeepSeekAPIKey 获取 DeepSeek API Key。
func GetDeepSeekAPIKey() string {
	return os.Getenv("DEEPSEEK_API_KEY")
}

// GetDeepSeekBaseURL 获取 DeepSeek API 基础地址。
func GetDeepSeekBaseURL() string {
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return baseURL
}

// GetDeepSeekModel 获取用于提示词优化的 DeepSeek 模型名。
func GetDeepSeekModel() string {
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	return model
}

// GetPromptOptimizeCredits 获取提示词优化扣费钻石数。
func GetPromptOptimizeCredits() int {
	value := os.Getenv("PROMPT_OPTIMIZE_CREDITS")
	if value == "" {
		return 1
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 1
	}
	return parsed
}

// GetVolcengineAPIKey 获取火山引擎 API Key。
func GetVolcengineAPIKey() string {
	return os.Getenv("ARK_API_KEY")
}

// GetJWTSecret 获取 JWT 密钥。
func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

// GetLinuxDoClientID 获取 Linux.do OAuth Client ID。
func GetLinuxDoClientID() string {
	return os.Getenv("LINUXDO_CLIENT_ID")
}

// GetLinuxDoClientSecret 获取 Linux.do OAuth Client Secret。
func GetLinuxDoClientSecret() string {
	return os.Getenv("LINUXDO_CLIENT_SECRET")
}

// GetOAuthRedirectURL 获取 OAuth 回调地址。
func GetOAuthRedirectURL() string {
	return os.Getenv("OAUTH_REDIRECT_URL")
}

// GetLinuxDoCreditPID 获取 Linux.do Credit 商户 ID。
func GetLinuxDoCreditPID() string {
	return os.Getenv("LINUXDO_CREDIT_PID")
}

// GetLinuxDoCreditKey 获取 Linux.do Credit 商户密钥。
func GetLinuxDoCreditKey() string {
	return os.Getenv("LINUXDO_CREDIT_KEY")
}

// GetLinuxDoCreditNotifyURL 获取 Linux.do Credit 回调地址。
func GetLinuxDoCreditNotifyURL() string {
	return os.Getenv("LINUXDO_CREDIT_NOTIFY_URL")
}

// GetLinuxDoCreditReturnURL 获取 Linux.do Credit 支付完成跳转地址。
func GetLinuxDoCreditReturnURL() string {
	return os.Getenv("LINUXDO_CREDIT_RETURN_URL")
}

// GetAppEnv 获取应用运行环境。
func GetAppEnv() string {
	return os.Getenv("APP_ENV")
}

// GetCORSOrigins 获取 CORS 白名单来源。
func GetCORSOrigins() string {
	return os.Getenv("CORS_ORIGINS")
}

// GetPort 获取服务端口。
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8092"
	}
	return port
}
