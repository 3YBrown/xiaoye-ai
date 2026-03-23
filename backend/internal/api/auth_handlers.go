package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google-ai-proxy/internal/auth"
	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/email"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SendVerificationCode 发送验证码
func SendVerificationCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式无效"})
		return
	}

	if req.Type != "register" && req.Type != "login" && req.Type != "reset" && req.Type != "bind" {
		req.Type = "register"
	}

	// 检查邮箱是否已注册
	var existingUser db.User
	result := db.DB.Where("email = ?", req.Email).First(&existingUser)

	if (req.Type == "register" || req.Type == "bind") && result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已注册，请直接登录"})
		return
	}

	if (req.Type == "login" || req.Type == "reset") && result.Error == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "该邮箱未注册，请先注册"})
		return
	}

	// 检查频率限制 (1分钟内只能发送1次)
	var recentCode db.EmailVerification
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	if db.DB.Where("email = ? AND created_at > ?", req.Email, oneMinuteAgo).First(&recentCode).Error == nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请稍后再试，验证码发送过于频繁"})
		return
	}

	// 生成验证码
	code := db.GenerateVerificationCode()

	// 保存到数据库
	verification := db.EmailVerification{
		Email:     req.Email,
		Code:      code,
		Type:      req.Type,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
		CreatedAt: time.Now(),
	}
	if err := db.DB.Create(&verification).Error; err != nil {
		log.Printf("保存验证码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证码失败"})
		return
	}

	// 发送邮件
	if err := email.SendVerificationCode(req.Email, code, req.Type); err != nil {
		log.Printf("发送验证码邮件失败: %v", err)
		// 即使邮件发送失败，在开发环境下也返回成功
		if !email.IsConfigured() {
			log.Printf("开发模式: 验证码 [%s] 发送到 [%s]", code, req.Email)
			resp := gin.H{"message": "验证码已发送 (开发模式)"}
			if config.GetAppEnv() != "production" {
				resp["dev_code"] = code // 仅非生产环境返回验证码
			}
			c.JSON(http.StatusOK, resp)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送到您的邮箱"})
}

// Register 用户注册
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式无效"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少6位"})
		return
	}

	if len(req.Code) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码格式无效"})
		return
	}

	// 验证验证码
	var verification db.EmailVerification
	result := db.DB.Where(
		"email = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
		req.Email, req.Code, "register", false, time.Now(),
	).Order("created_at DESC").First(&verification)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	// 检查邮箱是否已存在
	var existingUser db.User
	if db.DB.Where("email = ?", req.Email).First(&existingUser).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已注册"})
		return
	}

	// 检查邀请码并获取邀请人
	var inviter *db.User
	if req.InviteCode != "" {
		req.InviteCode = strings.TrimSpace(strings.ToUpper(req.InviteCode))
		var inviterUser db.User
		if err := db.DB.Where("invite_code = ?", req.InviteCode).First(&inviterUser).Error; err == nil {
			inviter = &inviterUser
		} else {
			log.Printf("邀请码无效: %s", req.InviteCode)
			// 邀请码无效不阻止注册，只是不给邀请奖励
		}
	}

	// 密码加密
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 生成昵称
	nickname := req.Nickname
	if nickname == "" {
		// 使用邮箱前缀作为默认昵称
		parts := strings.Split(req.Email, "@")
		nickname = parts[0]
	}

	// 生成邀请码
	inviteCode := db.GenerateInviteCode()
	// 确保邀请码唯一
	for {
		var count int64
		db.DB.Model(&db.User{}).Where("invite_code = ?", inviteCode).Count(&count)
		if count == 0 {
			break
		}
		inviteCode = db.GenerateInviteCode()
	}

	// 创建用户
	now := time.Now()
	user := db.User{
		Email:         req.Email,
		PasswordHash:  passwordHash,
		Nickname:      nickname,
		Credits:       10, // 新用户赠送10钻石
		EmailVerified: true,
		InviteCode:    inviteCode,
		LastLoginAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 如果有邀请人，记录邀请关系
	if inviter != nil {
		user.InvitedBy = &inviter.ID
	}

	// 开启事务
	tx := db.DB.Begin()

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		log.Printf("创建用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 记录新用户注册赠送钻石流水
	if user.Credits > 0 {
		if err := recordCreditTransaction(
			tx,
			user.ID,
			user.Credits,
			CreditTxTypeRegisterGift,
			"register",
			fmt.Sprintf("%d", user.ID),
			"new user gift",
		); err != nil {
			tx.Rollback()
			log.Printf("记录注册赠送流水失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
			return
		}
	}

	// 如果有邀请人，给邀请人增加钻石并记录
	if inviter != nil {
		const inviteReward = 10      // 邀请奖励钻石
		const maxInviteCredits = 500 // 邀请钻石上限

		// 计算邀请人已获得的邀请钻石
		var totalInviteCredits int
		db.DB.Model(&db.InvitationRecord{}).
			Where("inviter_id = ?", inviter.ID).
			Select("COALESCE(SUM(credits_rewarded), 0)").
			Scan(&totalInviteCredits)

		// 检查是否已达到上限
		if totalInviteCredits >= maxInviteCredits {
			log.Printf("用户 %d 邀请钻石已达上限 %d，不再发放奖励", inviter.ID, maxInviteCredits)
		} else {
			// 计算实际可获得的钻石（防止超过上限）
			actualReward := inviteReward
			if totalInviteCredits+inviteReward > maxInviteCredits {
				actualReward = maxInviteCredits - totalInviteCredits
			}

			// 增加邀请人钻石和邀请数量
			if err := tx.Model(inviter).Updates(map[string]interface{}{
				"credits":      gorm.Expr("credits + ?", actualReward),
				"invite_count": gorm.Expr("invite_count + ?", 1),
				"updated_at":   now,
			}).Error; err != nil {
				tx.Rollback()
				log.Printf("更新邀请人钻石失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
				return
			}

			// 创建邀请记录
			invitationRecord := db.InvitationRecord{
				InviterID:       inviter.ID,
				InviteeID:       user.ID,
				InviteeEmail:    user.Email,
				CreditsRewarded: actualReward,
				CreatedAt:       now,
			}
			if err := tx.Create(&invitationRecord).Error; err != nil {
				tx.Rollback()
				log.Printf("创建邀请记录失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
				return
			}

			if err := recordCreditTransaction(
				tx,
				inviter.ID,
				actualReward,
				CreditTxTypeInviteReward,
				"invitation",
				fmt.Sprintf("%d", invitationRecord.ID),
				fmt.Sprintf("invite user %d", user.ID),
			); err != nil {
				tx.Rollback()
				log.Printf("记录邀请奖励流水失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
				return
			}

			log.Printf("邀请奖励: 用户 %d 邀请 %d 成功，获得 %d 钻石 (累计: %d/%d)",
				inviter.ID, user.ID, actualReward, totalInviteCredits+actualReward, maxInviteCredits)
		}
	}

	tx.Commit()

	// 发送欢迎通知
	welcomeNotification := db.UserNotification{
		UserID:    user.ID,
		BizKey:    fmt.Sprintf("welcome-%d", user.ID),
		Title:     "欢迎来到 小野 AI ✨",
		Summary:   "注册成功！你已获得 10 钻石，每日签到可免费领取更多钻石。",
		Content:   "Hi，欢迎加入 小野 AI！🎉\n\n你已获得 10 颗钻石作为新人礼物，可以立即开始 AI 创作。\n\n💡 几个小贴士帮你快速上手：\n\n1. **免费获取钻石** — 每日签到即可领取钻石，连续签到奖励更多\n2. **开始创作** — 使用 BananaPro 模型，输入提示词即可生成精美图片和视频\n3. **灵感广场** — 浏览其他创作者的作品，一键「做同款」\n4. **加入社群** — 微信扫码加入 AI 创作者社群，交流技巧、获取免费额度\n\n祝你创作愉快！",
		IsRead:    false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.DB.Create(&welcomeNotification).Error; err != nil {
		log.Printf("创建欢迎通知失败: %v", err)
		// 不影响注册流程，仅记录日志
	}

	// 标记验证码已使用
	db.DB.Model(&verification).Update("used", true)

	// 生成JWT令牌
	token, err := auth.GenerateUserToken(user.ID, user.Email)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册成功但登录失败"})
		return
	}

	log.Printf("用户注册成功: %s (ID: %d), 邀请码: %s", user.Email, user.ID, user.InviteCode)

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"token":   token,
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"nickname":    user.Nickname,
			"credits":     user.Credits,
			"invite_code": user.InviteCode,
		},
	})
}

// ResetPassword 重置密码
func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式无效"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少6位"})
		return
	}

	if len(req.Code) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码格式无效"})
		return
	}

	// 验证验证码
	var verification db.EmailVerification
	result := db.DB.Where(
		"email = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
		req.Email, req.Code, "reset", false, time.Now(),
	).Order("created_at DESC").First(&verification)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	// 查找用户
	var user db.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 密码加密
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	// 更新密码
	if err := db.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash": passwordHash,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		log.Printf("更新密码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	// 标记验证码已使用
	db.DB.Model(&verification).Update("used", true)

	log.Printf("用户密码重置成功: %s (ID: %d)", user.Email, user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功，请使用新密码登录"})
}

// LoginWithEmail 邮箱登录（支持密码和验证码）
func LoginWithEmail(c *gin.Context) {
	var req LoginWithEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式无效"})
		return
	}

	// 查找用户
	var user db.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		}
		return
	}

	// 检查用户状态
	if user.Status != "" && user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return
	}

	// 验证方式：密码或验证码
	if req.Password != "" {
		// 密码登录
		if !auth.CheckPassword(req.Password, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
			return
		}
	} else if req.Code != "" {
		// 验证码登录
		var verification db.EmailVerification
		result := db.DB.Where(
			"email = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
			req.Email, req.Code, "login", false, time.Now(),
		).Order("created_at DESC").First(&verification)

		if result.Error != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "验证码无效或已过期"})
			return
		}
		// 标记验证码已使用
		db.DB.Model(&verification).Update("used", true)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供密码或验证码"})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	db.DB.Model(&user).Updates(map[string]interface{}{
		"last_login_at": now,
		"updated_at":    now,
	})

	// 生成JWT令牌
	token, err := auth.GenerateUserToken(user.ID, user.Email)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		return
	}

	log.Printf("用户登录成功: %s (ID: %d)", user.Email, user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"nickname": user.Nickname,
			"credits":  user.Credits,
		},
	})
}

// RedeemKey 兑换密钥获取钻石
func RedeemKey(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var req RedeemKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密钥"})
		return
	}

	// 验证密钥
	claims, err := auth.ValidateLicenseKey(req.Key)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密钥无效"})
		return
	}

	// 查找或创建License记录
	var license db.License
	result := db.DB.First(&license, "id = ?", claims.ID)

	if result.Error == gorm.ErrRecordNotFound {
		// 首次兑换，创建License记录
		now := time.Now()
		license = db.License{
			ID:          claims.ID,
			Balance:     0, // 初始余额为0，钻石将直接加到用户账户
			Status:      db.LicenseStatusRedeemed,
			OriginalKey: req.Key,
			RedeemedBy:  &userID,
			RedeemedAt:  &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := db.DB.Create(&license).Error; err != nil {
			log.Printf("创建License记录失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
			return
		}
	} else if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
		return
	} else {
		// 检查是否已被兑换
		if license.Status == db.LicenseStatusRedeemed {
			c.JSON(http.StatusConflict, gin.H{"error": "该密钥已被兑换"})
			return
		}
		if license.Status == db.LicenseStatusDisabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "该密钥已被禁用"})
			return
		}

		// 如果之前是通过老方式登录使用的，也需要处理
		// 将剩余余额加到用户账户，然后标记为已兑换
		claims.Credits = license.Balance // 使用数据库中的余额而不是token中的
	}

	// 获取用户
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	// 开启事务
	tx := db.DB.Begin()

	// 增加用户钻石
	creditsToAdd := claims.Credits
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"credits":        gorm.Expr("credits + ?", creditsToAdd),
		"total_redeemed": gorm.Expr("total_redeemed + ?", creditsToAdd),
		"updated_at":     time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("更新用户钻石失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
		return
	}

	if err := recordCreditTransaction(
		tx,
		userID,
		creditsToAdd,
		CreditTxTypeRedeem,
		"license",
		claims.ID,
		"redeem key",
	); err != nil {
		tx.Rollback()
		log.Printf("记录兑换流水失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
		return
	}

	// 更新License状态
	now := time.Now()
	if err := tx.Model(&license).Updates(map[string]interface{}{
		"status":      db.LicenseStatusRedeemed,
		"balance":     0,
		"redeemed_by": userID,
		"redeemed_at": now,
		"updated_at":  now,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("更新License状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
		return
	}

	tx.Commit()

	// 重新获取用户最新钻石
	db.DB.First(&user, userID)

	log.Printf("用户 %d 兑换密钥成功，获得 %d 钻石，当前总钻石: %d", userID, creditsToAdd, user.Credits)

	c.JSON(http.StatusOK, gin.H{
		"message":         "兑换成功",
		"credits_added":   creditsToAdd,
		"current_credits": user.Credits,
	})
}

// GetUserMe 获取当前用户信息
func GetUserMe(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 如果用户没有邀请码，生成一个
	if user.InviteCode == "" {
		inviteCode := db.GenerateInviteCode()
		for {
			var count int64
			db.DB.Model(&db.User{}).Where("invite_code = ?", inviteCode).Count(&count)
			if count == 0 {
				break
			}
			inviteCode = db.GenerateInviteCode()
		}
		db.DB.Model(&user).Update("invite_code", inviteCode)
		user.InviteCode = inviteCode
	}

	// Compute checkin status
	today := time.Now().Format("2006-01-02")
	checkinAvailable := true
	if user.LastCheckinDate != nil && user.LastCheckinDate.Format("2006-01-02") == today {
		checkinAvailable = false
	}
	nextReward := calcCheckinReward(nextCheckinStreak(user))

	c.JSON(http.StatusOK, gin.H{
		"id":                      user.ID,
		"email":                   user.Email,
		"nickname":                user.Nickname,
		"avatar":                  user.Avatar,
		"credits":                 user.Credits,
		"total_redeemed":          user.TotalRedeemed,
		"usage_count":             user.UsageCount,
		"invite_code":             user.InviteCode,
		"invite_count":            user.InviteCount,
		"created_at":              user.CreatedAt,
		"last_login_at":           user.LastLoginAt,
		"daily_checkin_available": checkinAvailable,
		"checkin_streak":          user.CheckinStreak,
		"next_checkin_reward":     nextReward,
		"is_linuxdo":              user.LinuxDoID != nil,
		"email_verified":          user.EmailVerified,
	})
}

// GetInvitationRecords 获取邀请记录
func GetInvitationRecords(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	// 获取用户信息
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 如果用户没有邀请码，生成一个
	if user.InviteCode == "" {
		inviteCode := db.GenerateInviteCode()
		for {
			var count int64
			db.DB.Model(&db.User{}).Where("invite_code = ?", inviteCode).Count(&count)
			if count == 0 {
				break
			}
			inviteCode = db.GenerateInviteCode()
		}
		db.DB.Model(&user).Update("invite_code", inviteCode)
		user.InviteCode = inviteCode
	}

	// 获取邀请记录
	var records []db.InvitationRecord
	db.DB.Where("inviter_id = ?", userID).Order("created_at DESC").Find(&records)

	// 计算总获得钻石
	var totalCredits int
	for _, r := range records {
		totalCredits += r.CreditsRewarded
	}

	// 格式化返回数据
	recordList := make([]gin.H, len(records))
	for i, r := range records {
		// 隐藏邮箱部分信息
		emailParts := strings.Split(r.InviteeEmail, "@")
		maskedEmail := r.InviteeEmail
		if len(emailParts) == 2 && len(emailParts[0]) > 2 {
			maskedEmail = emailParts[0][:2] + "***@" + emailParts[1]
		}

		recordList[i] = gin.H{
			"id":               r.ID,
			"invitee_email":    maskedEmail,
			"credits_rewarded": r.CreditsRewarded,
			"created_at":       r.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"invite_code":   user.InviteCode,
		"invite_count":  user.InviteCount,
		"total_credits": totalCredits,
		"records":       recordList,
	})
}

// GetCreditTransactions 获取用户钻石流水
func GetCreditTransactions(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	txType := strings.TrimSpace(c.Query("type"))

	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := db.DB.Model(&db.CreditTransaction{}).Where("user_id = ?", userID)
	if txType != "" && txType != "all" {
		query = query.Where("type = ?", txType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询钻石流水失败"})
		return
	}

	var rows []db.CreditTransaction
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询钻石流水失败"})
		return
	}

	items := make([]gin.H, len(rows))
	for i, row := range rows {
		items[i] = gin.H{
			"id":            row.ID,
			"delta":         row.Delta,
			"balance_after": row.BalanceAfter,
			"type":          row.Type,
			"source":        row.Source,
			"source_id":     row.SourceID,
			"note":          row.Note,
			"created_at":    row.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": items,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// localDateStr returns "2006-01-02" in local timezone for t.
func localDateStr(t time.Time) string {
	return t.Format("2006-01-02")
}

// yesterdayDateStr returns yesterday's date string in local timezone.
func yesterdayDateStr() string {
	return localDateStr(time.Now().AddDate(0, 0, -1))
}

// calcCheckinReward returns the diamond reward for a given streak day (1-7 cycle).
func calcCheckinReward(streak int) int {
	day := ((streak - 1) % 7) + 1 // 1..7
	switch day {
	case 7:
		return 15
	default:
		return 4 + day // day1=5, day2=6, ..., day6=10
	}
}

// nextCheckinStreak computes what the streak would be on the next checkin.
func nextCheckinStreak(user db.User) int {
	if user.LastCheckinDate == nil {
		return 1
	}
	if localDateStr(*user.LastCheckinDate) == yesterdayDateStr() {
		return user.CheckinStreak + 1
	}
	return 1
}

// DailyCheckin 每日签到领取钻石
func DailyCheckin(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	now := time.Now()
	todayStr := localDateStr(now)
	yesterdayStr := localDateStr(now.AddDate(0, 0, -1))

	// Use transaction with row lock to prevent race conditions
	tx := db.DB.Begin()
	if tx.Error != nil {
		log.Printf("[Checkin] 启动事务失败 [用户:%d]: %v", userID, tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	var user db.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// Check if already checked in today
	if user.LastCheckinDate != nil && localDateStr(*user.LastCheckinDate) == todayStr {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "今日已签到"})
		return
	}

	// Determine new streak
	var newStreak int
	if user.LastCheckinDate != nil && localDateStr(*user.LastCheckinDate) == yesterdayStr {
		newStreak = user.CheckinStreak + 1
	} else {
		newStreak = 1
	}

	reward := calcCheckinReward(newStreak)

	if err := tx.Model(&user).Updates(map[string]interface{}{
		"credits":           gorm.Expr("credits + ?", reward),
		"checkin_streak":    newStreak,
		"last_checkin_date": todayStr,
		"updated_at":        now,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[Checkin] 更新用户失败 [用户:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	if err := recordCreditTransaction(
		tx,
		userID,
		reward,
		CreditTxTypeDailyCheckin,
		"daily_checkin",
		fmt.Sprintf("streak_%d", newStreak),
		fmt.Sprintf("daily checkin day %d", ((newStreak-1)%7)+1),
	); err != nil {
		tx.Rollback()
		log.Printf("[Checkin] 记录流水失败 [用户:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[Checkin] 提交事务失败 [用户:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	// Fetch updated credits
	db.DB.Select("credits").First(&user, userID)

	log.Printf("[Checkin] 签到成功 [用户:%d, streak:%d, reward:%d]", userID, newStreak, reward)

	c.JSON(http.StatusOK, gin.H{
		"credits_added":   reward,
		"current_credits": user.Credits,
		"checkin_streak":  newStreak,
	})
}

// BindEmail 绑定邮箱（OAuth 用户）
func BindEmail(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var req BindEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式无效"})
		return
	}

	if len(req.Code) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码格式无效"})
		return
	}

	// 获取当前用户，检查是否需要绑定
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已验证，无需绑定"})
		return
	}

	// 验证验证码
	var verification db.EmailVerification
	result := db.DB.Where(
		"email = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?",
		req.Email, req.Code, "bind", false, time.Now(),
	).Order("created_at DESC").First(&verification)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	// 事务：检查邮箱唯一性 + 更新
	tx := db.DB.Begin()

	var existingUser db.User
	if tx.Where("email = ?", req.Email).First(&existingUser).Error == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被其他用户注册"})
		return
	}

	if err := tx.Model(&user).Updates(map[string]interface{}{
		"email":          req.Email,
		"email_verified": true,
		"updated_at":     time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("绑定邮箱失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定邮箱失败"})
		return
	}

	// 标记验证码已使用
	tx.Model(&verification).Update("used", true)

	if err := tx.Commit().Error; err != nil {
		log.Printf("绑定邮箱事务提交失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定邮箱失败"})
		return
	}

	// 重新读取用户
	db.DB.First(&user, userID)

	log.Printf("用户 %d 绑定邮箱成功: %s", userID, req.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "邮箱绑定成功",
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"nickname":       user.Nickname,
			"avatar":         user.Avatar,
			"credits":        user.Credits,
			"invite_code":    user.InviteCode,
			"email_verified": user.EmailVerified,
		},
	})
}

// StartVerificationCleanup 启动后台定期清理过期/已使用的验证码
func StartVerificationCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			result := db.DB.Where("used = ? OR expires_at < ?", true, time.Now()).Delete(&db.EmailVerification{})
			if result.RowsAffected > 0 {
				log.Printf("[验证码清理] 已清理 %d 条过期/已使用的验证码记录", result.RowsAffected)
			}
		}
	}()
	log.Println("[验证码清理] 后台清理 Worker 已启动 (每小时执行)")
}
