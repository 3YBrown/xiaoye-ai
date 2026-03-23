package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// sanitizeRequestBody 移除请求体中的 image 字段以减少日志大小
func sanitizeRequestBody(reqBody interface{}) interface{} {
	if reqBody == nil {
		return nil
	}

	reqBytes, _ := json.Marshal(reqBody)
	var v interface{}
	if err := json.Unmarshal(reqBytes, &v); err != nil {
		return reqBody
	}

	return sanitizeAny(v)
}

var redactedRequestKeys = map[string]struct{}{
	"image":         {},
	"images":        {},
	"input_images":  {},
	"output_images": {},
	"inputImages":   {},
	"outputImages":  {},
}

func sanitizeAny(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			if _, ok := redactedRequestKeys[k]; ok {
				out[k] = summarizeRedacted(vv)
				continue
			}
			out[k] = sanitizeAny(vv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(t))
		for _, vv := range t {
			out = append(out, sanitizeAny(vv))
		}
		return out
	default:
		return v
	}
}

func summarizeRedacted(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return map[string]interface{}{"redacted": true, "length": len(t)}
	case []interface{}:
		return map[string]interface{}{"redacted": true, "count": len(t)}
	default:
		return map[string]interface{}{"redacted": true}
	}
}

// logAPICall 异步记录 API 调用，不会影响主流程
// 即使记录失败也只会输出日志，不会影响用户请求的响应
func logAPICall(endpoint string, reqBody interface{}, respCode int, respBody interface{}, duration time.Duration, userID string) {
	// 在独立的 goroutine 中异步执行日志记录，不阻塞主流程
	go func() {
		// 使用 defer recover 捕获任何 panic，防止 goroutine 崩溃
		defer func() {
			if r := recover(); r != nil {
				log.Printf("警告: API 日志记录发生 panic - %v", r)
			}
		}()

		// 清理请求体（移除 image 字段）
		sanitizedReqBody := sanitizeRequestBody(reqBody)

		// 构建日志对象
		reqBytes, _ := json.Marshal(sanitizedReqBody)
		respBytes, _ := json.Marshal(respBody)

		apiLog := db.APILog{
			UserID:       userID,
			Endpoint:     endpoint,
			RequestBody:  string(reqBytes),
			ResponseBody: string(respBytes),
			ResponseCode: respCode,
			DurationMs:   int(duration.Milliseconds()),
			CreatedAt:    time.Now(),
		}

		// 写入数据库，失败只输出日志不影响主流程
		if err := db.DB.Create(&apiLog).Error; err != nil {
			log.Printf("警告: 无法记录 API 日志 - %v", err)
		}
	}()
}

// getActiveUser 获取并验证用户状态，失败时自动写入 HTTP 响应
func getActiveUser(c *gin.Context, userID uint64) (*db.User, bool) {
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return nil, false
	}
	if user.Status != "" && user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return nil, false
	}
	return &user, true
}

const (
	CreditTxTypeRegisterGift       = "register_gift"
	CreditTxTypeInviteReward       = "invite_reward"
	CreditTxTypeRedeem             = "redeem"
	CreditTxTypeGenerateCost       = "generate_cost"
	CreditTxTypePromptOptimizeCost = "prompt_optimize_cost"
	CreditTxTypeRefund             = "refund"
	CreditTxTypeDailyCheckin       = "daily_checkin"
	CreditTxTypeOnlinePayment      = "online_payment"
)

// recordCreditTransaction 在用户余额已更新后写入流水。
func recordCreditTransaction(tx *gorm.DB, userID uint64, delta int, txType, source, sourceID, note string) error {
	if delta == 0 {
		return nil
	}

	var user db.User
	if err := tx.Select("id", "credits").First(&user, userID).Error; err != nil {
		return err
	}

	record := db.CreditTransaction{
		UserID:       userID,
		Delta:        delta,
		BalanceAfter: user.Credits,
		Type:         txType,
		Source:       source,
		SourceID:     sourceID,
		Note:         note,
		CreatedAt:    time.Now(),
	}
	return tx.Create(&record).Error
}

func parseRefundSource(reason string) (string, string, string) {
	source := "generate"
	sourceID := ""
	note := reason

	if strings.HasPrefix(reason, "video-task-") {
		sourceID = strings.TrimPrefix(reason, "video-task-")
	}
	if strings.HasPrefix(reason, "prompt-optimize-") {
		source = "prompt_optimize"
		sourceID = strings.TrimPrefix(reason, "prompt-optimize-")
	}
	if strings.HasPrefix(reason, "reverse-prompt-") {
		source = "reverse_prompt"
		sourceID = strings.TrimPrefix(reason, "reverse-prompt-")
	}
	return source, sourceID, note
}

// refundCredits 生成失败时退还钻石（统一退款函数）
func refundCredits(userID uint64, credits int, reason string) {
	if credits <= 0 {
		return
	}

	tx := db.DB.Begin()
	if tx.Error != nil {
		log.Printf("[Refund] 启动事务失败 [用户:%d, 原因:%s, 钻石:%d]: %v", userID, reason, credits, tx.Error)
		return
	}

	result := tx.Exec("UPDATE users SET credits = credits + ?, updated_at = NOW() WHERE id = ?", credits, userID)
	if result.Error != nil || result.RowsAffected == 0 {
		tx.Rollback()
		if result.Error != nil {
			log.Printf("[Refund] 退还钻石失败 [用户:%d, 原因:%s, 钻石:%d]: %v", userID, reason, credits, result.Error)
		} else {
			log.Printf("[Refund] 退还钻石失败 [用户:%d, 原因:%s, 钻石:%d]: rows=0", userID, reason, credits)
		}
		return
	}

	source, sourceID, note := parseRefundSource(reason)
	if err := recordCreditTransaction(tx, userID, credits, CreditTxTypeRefund, source, sourceID, note); err != nil {
		tx.Rollback()
		log.Printf("[Refund] 记录退款流水失败 [用户:%d, 原因:%s, 钻石:%d]: %v", userID, reason, credits, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[Refund] 提交事务失败 [用户:%d, 原因:%s, 钻石:%d]: %v", userID, reason, credits, err)
		return
	}

	log.Printf("[Refund] 钻石已退还 [用户:%d, 原因:%s, 钻石:%d]", userID, reason, credits)
}
