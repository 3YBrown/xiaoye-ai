package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/payment"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Payment plans: plan name → (credits amount as string, diamonds count)
var paymentPlans = map[string]struct {
	Amount   string
	Diamonds int
}{
	"starter": {"129", 100},
	"popular": {"559", 500},
	"pro":     {"999", 1000},
}

var linuxdoProvider = payment.NewLinuxDoCreditProvider()

// generateOrderNo creates a unique order number: NB + timestamp + 6-digit random.
// Retries if a collision is detected in the database.
func generateOrderNo() string {
	for i := 0; i < 5; i++ {
		no := fmt.Sprintf("NB%s%06d",
			time.Now().Format("20060102150405"),
			rand.Intn(1000000))
		var count int64
		db.DB.Model(&db.PaymentOrder{}).Where("order_no = ?", no).Count(&count)
		if count == 0 {
			return no
		}
	}
	// Fallback with nanosecond precision
	return fmt.Sprintf("NB%d%06d", time.Now().UnixNano(), rand.Intn(1000000))
}

// amountsEqual compares two money amount strings numerically (e.g. "10" == "10.00").
func amountsEqual(a, b string) bool {
	fa, errA := strconv.ParseFloat(a, 64)
	fb, errB := strconv.ParseFloat(b, 64)
	if errA != nil || errB != nil {
		return a == b
	}
	return math.Abs(fa-fb) < 0.001
}

// CreatePaymentOrder handles POST /api/user/payment/create
func CreatePaymentOrder(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var req struct {
		Plan string `json:"plan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	plan, ok := paymentPlans[req.Plan]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的套餐"})
		return
	}

	// Verify the user is a linux.do user
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.LinuxDoID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅 Linux.do 用户可使用积分支付"})
		return
	}

	orderNo := generateOrderNo()
	now := time.Now()
	order := db.PaymentOrder{
		UserID:    userID,
		OrderNo:   orderNo,
		Provider:  linuxdoProvider.Name(),
		Amount:    plan.Amount,
		Diamonds:  plan.Diamonds,
		PlanName:  req.Plan,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := db.DB.Create(&order).Error; err != nil {
		log.Printf("[Payment] 创建订单失败 [用户:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建订单失败"})
		return
	}

	result, err := linuxdoProvider.CreatePayment(&order)
	if err != nil {
		log.Printf("[Payment] 生成支付链接失败 [订单:%s]: %v", orderNo, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成支付链接失败"})
		return
	}

	log.Printf("[Payment] 订单已创建 [用户:%d, 订单:%s, 套餐:%s, 积分:%s, 钻石:%d]",
		userID, orderNo, req.Plan, plan.Amount, plan.Diamonds)

	c.JSON(http.StatusOK, gin.H{
		"order_no":    orderNo,
		"payment_url": result.PaymentURL,
	})
}

// fulfillPaymentOrder processes a paid order inside a transaction:
// locks the order row, marks it paid, adds diamonds to the user, and records
// the credit transaction. Returns true if the order was newly fulfilled,
// false if it was already processed. An error is returned on failure.
func fulfillPaymentOrder(orderNo, tradeNo, notifyData string) (bool, error) {
	tx := db.DB.Begin()
	if tx.Error != nil {
		return false, fmt.Errorf("begin tx: %w", tx.Error)
	}

	// Lock order row
	var order db.PaymentOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("lock order: %w", err)
	}

	// Already processed (idempotent)
	if order.Status != "pending" {
		tx.Rollback()
		return false, nil
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":            "paid",
		"provider_trade_no": tradeNo,
		"paid_at":           now,
		"updated_at":        now,
	}
	if notifyData != "" {
		updates["notify_data"] = notifyData
	}
	if err := tx.Model(&order).Updates(updates).Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("update order: %w", err)
	}

	// Add diamonds to user
	if err := tx.Model(&db.User{}).Where("id = ?", order.UserID).Updates(map[string]interface{}{
		"credits":        gorm.Expr("credits + ?", order.Diamonds),
		"total_redeemed": gorm.Expr("total_redeemed + ?", order.Diamonds),
		"updated_at":     now,
	}).Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("add diamonds: %w", err)
	}

	// Record credit transaction
	if err := recordCreditTransaction(
		tx,
		order.UserID,
		order.Diamonds,
		CreditTxTypeOnlinePayment,
		"linuxdo_credit",
		order.OrderNo,
		fmt.Sprintf("online payment %s %s credits", order.PlanName, order.Amount),
	); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("record tx: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}

	log.Printf("[Payment] 支付成功 [用户:%d, 订单:%s, 钻石:+%d]",
		order.UserID, orderNo, order.Diamonds)
	return true, nil
}

// LinuxDoCreditNotify handles GET /api/payment/notify/linuxdo (callback from credit.linux.do)
func LinuxDoCreditNotify(c *gin.Context) {
	// Collect all query params
	params := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// Store raw notify data for debugging
	notifyJSON, _ := json.Marshal(params)
	log.Printf("[Payment] 收到回调: %s", string(notifyJSON))

	// Verify signature
	ok, err := linuxdoProvider.VerifyNotify(params)
	if err != nil || !ok {
		log.Printf("[Payment] 签名验证失败: ok=%v, err=%v", ok, err)
		c.String(http.StatusBadRequest, "sign error")
		return
	}

	outTradeNo := params["out_trade_no"]
	tradeNo := params["trade_no"]
	tradeStatus := params["trade_status"]
	money := params["money"]

	if outTradeNo == "" {
		c.String(http.StatusBadRequest, "missing out_trade_no")
		return
	}

	// Find order
	var order db.PaymentOrder
	if err := db.DB.Where("order_no = ?", outTradeNo).First(&order).Error; err != nil {
		log.Printf("[Payment] 订单不存在 [out_trade_no:%s]: %v", outTradeNo, err)
		c.String(http.StatusNotFound, "order not found")
		return
	}

	// Non-success: just save notify data and return
	if tradeStatus != "TRADE_SUCCESS" {
		db.DB.Model(&order).Updates(map[string]interface{}{
			"notify_data":       string(notifyJSON),
			"provider_trade_no": tradeNo,
			"updated_at":        time.Now(),
		})
		log.Printf("[Payment] 非成功状态 [订单:%s, status:%s]", outTradeNo, tradeStatus)
		c.String(http.StatusOK, "success")
		return
	}

	// Already processed (idempotent)
	if order.Status != "pending" {
		log.Printf("[Payment] 订单已处理 [订单:%s, 当前状态:%s]", outTradeNo, order.Status)
		c.String(http.StatusOK, "success")
		return
	}

	// Verify amount matches (numeric comparison to handle "10" vs "10.00")
	if !amountsEqual(money, order.Amount) {
		log.Printf("[Payment] 金额不匹配 [订单:%s, 期望:%s, 实际:%s]", outTradeNo, order.Amount, money)
		c.String(http.StatusBadRequest, "amount mismatch")
		return
	}

	fulfilled, err := fulfillPaymentOrder(outTradeNo, tradeNo, string(notifyJSON))
	if err != nil {
		log.Printf("[Payment] 处理订单失败 [订单:%s]: %v", outTradeNo, err)
		c.String(http.StatusInternalServerError, "fail")
		return
	}
	if !fulfilled {
		log.Printf("[Payment] 订单已被其他回调处理 [订单:%s]", outTradeNo)
	}

	c.String(http.StatusOK, "success")
}

// GetPaymentStatus handles GET /api/user/payment/status/:orderNo
func GetPaymentStatus(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	orderNo := c.Param("orderNo")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少订单号"})
		return
	}

	var order db.PaymentOrder
	if err := db.DB.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// If still pending, try querying the provider and fulfill if paid
	if order.Status == "pending" {
		result, err := linuxdoProvider.QueryOrder(&order)
		if err == nil && result.TradeStatus == "TRADE_SUCCESS" {
			log.Printf("[Payment] 主动查询发现已支付 [订单:%s], 触发处理", orderNo)
			fulfilled, fErr := fulfillPaymentOrder(order.OrderNo, result.TradeNo, "")
			if fErr != nil {
				log.Printf("[Payment] 主动查询后处理失败 [订单:%s]: %v", orderNo, fErr)
			} else if fulfilled {
				// Re-read order to get updated status
				db.DB.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&order)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"order_no":  order.OrderNo,
		"status":    order.Status,
		"diamonds":  order.Diamonds,
		"plan_name": order.PlanName,
		"amount":    order.Amount,
		"paid_at":   order.PaidAt,
	})
}

// GetPaymentOrders handles GET /api/user/payment/orders
func GetPaymentOrders(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	db.DB.Model(&db.PaymentOrder{}).Where("user_id = ?", userID).Count(&total)

	var orders []db.PaymentOrder
	db.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders)

	items := make([]gin.H, len(orders))
	for i, o := range orders {
		items[i] = gin.H{
			"order_no":   o.OrderNo,
			"provider":   o.Provider,
			"amount":     o.Amount,
			"diamonds":   o.Diamonds,
			"plan_name":  o.PlanName,
			"status":     o.Status,
			"paid_at":    o.PaidAt,
			"created_at": o.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
