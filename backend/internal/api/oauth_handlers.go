package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"google-ai-proxy/internal/auth"
	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// oauthStateStore stores state -> expiry for CSRF protection.
var oauthStateStore = struct {
	sync.Mutex
	states map[string]time.Time
}{states: make(map[string]time.Time)}

func init() {
	// Periodically clean expired states.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			oauthStateStore.Lock()
			now := time.Now()
			for k, v := range oauthStateStore.states {
				if now.After(v) {
					delete(oauthStateStore.states, k)
				}
			}
			oauthStateStore.Unlock()
		}
	}()
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)
	oauthStateStore.Lock()
	oauthStateStore.states[state] = time.Now().Add(10 * time.Minute)
	oauthStateStore.Unlock()
	return state, nil
}

func validateState(state string) bool {
	oauthStateStore.Lock()
	defer oauthStateStore.Unlock()
	expiry, ok := oauthStateStore.states[state]
	if !ok {
		return false
	}
	delete(oauthStateStore.states, state)
	return time.Now().Before(expiry)
}

func linuxDoOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.GetLinuxDoClientID(),
		ClientSecret: config.GetLinuxDoClientSecret(),
		RedirectURL:  config.GetOAuthRedirectURL(),
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://connect.linux.do/oauth2/authorize",
			TokenURL:  "https://connect.linux.do/oauth2/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		Scopes: []string{},
	}
}

// linuxDoUser represents the user info from linux.do API.
type linuxDoUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar_template"`
	Email    string `json:"email"`
}

// LinuxDoOAuthURL returns the authorization URL for linux.do OAuth.
func LinuxDoOAuthURL(c *gin.Context) {
	state, err := generateState()
	if err != nil {
		log.Printf("generate oauth state failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成授权链接失败"})
		return
	}

	cfg := linuxDoOAuthConfig()
	url := cfg.AuthCodeURL(state)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// LinuxDoOAuthCallback handles the OAuth callback from linux.do.
func LinuxDoOAuthCallback(c *gin.Context) {
	var req OAuthCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	// Validate state (CSRF protection)
	if !validateState(req.State) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的授权状态，请重新登录"})
		return
	}

	// Exchange code for token
	cfg := linuxDoOAuthConfig()
	token, err := cfg.Exchange(c.Request.Context(), req.Code)
	if err != nil {
		log.Printf("oauth token exchange failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "授权失败，请重试"})
		return
	}

	// Fetch user info from linux.do
	linuxDoUserInfo, err := fetchLinuxDoUser(token.AccessToken)
	if err != nil {
		log.Printf("fetch linux.do user info failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	linuxDoID := strconv.Itoa(linuxDoUserInfo.ID)

	// Look up existing user by linuxdo_id
	var user db.User
	result := db.DB.Where("linuxdo_id = ?", linuxDoID).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new user
		user, err = createOAuthUser(linuxDoID, linuxDoUserInfo)
		if err != nil {
			log.Printf("create oauth user failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
			return
		}
	} else if result.Error != nil {
		log.Printf("query user by linuxdo_id failed: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	// Check user status
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return
	}

	// Update last login time
	now := time.Now()
	db.DB.Model(&user).Update("last_login_at", &now)

	// Generate JWT
	jwtToken, err := auth.GenerateUserToken(user.ID, user.Email)
	if err != nil {
		log.Printf("generate jwt failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"nickname":       user.Nickname,
			"avatar":         user.Avatar,
			"credits":        user.Credits,
			"invite_code":    user.InviteCode,
			"status":         user.Status,
			"email_verified": user.EmailVerified,
		},
	})
}

var oauthHTTPClient = &http.Client{Timeout: 15 * time.Second}

func fetchLinuxDoUser(accessToken string) (*linuxDoUser, error) {
	req, err := http.NewRequest("GET", "https://connect.linux.do/api/user", nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var user linuxDoUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse user info failed: %w", err)
	}

	return &user, nil
}

func createOAuthUser(linuxDoID string, info *linuxDoUser) (db.User, error) {
	// Generate unique invite code
	var inviteCode string
	for i := 0; i < 10; i++ {
		inviteCode = db.GenerateInviteCode()
		var count int64
		db.DB.Model(&db.User{}).Where("invite_code = ?", inviteCode).Count(&count)
		if count == 0 {
			break
		}
	}

	// Use linux.do username or name as nickname
	nickname := info.Username
	if info.Name != "" {
		nickname = info.Name
	}

	// Build email: use linux.do email if available, otherwise generate a placeholder.
	// If the email already belongs to another user, fall back to a placeholder
	// to avoid unique constraint violation.
	userEmail := info.Email
	if userEmail != "" {
		var count int64
		db.DB.Model(&db.User{}).Where("email = ?", userEmail).Count(&count)
		if count > 0 {
			userEmail = ""
		}
	}
	isPlaceholderEmail := false
	if userEmail == "" {
		userEmail = fmt.Sprintf("linuxdo_%d@oauth.local", info.ID)
		isPlaceholderEmail = true
	}

	now := time.Now()
	user := db.User{
		Email:         userEmail,
		Nickname:      nickname,
		LinuxDoID:     &linuxDoID,
		Credits:       10,
		Status:        "active",
		EmailVerified: !isPlaceholderEmail,
		InviteCode:    inviteCode,
		LastLoginAt:   &now,
	}

	tx := db.DB.Begin()

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return db.User{}, fmt.Errorf("create user failed: %w", err)
	}

	// Record registration gift transaction
	creditTx := db.CreditTransaction{
		UserID:       user.ID,
		Delta:        10,
		BalanceAfter: 10,
		Type:         "register_gift",
		Source:       "oauth_linuxdo",
		Note:         "Linux.do OAuth 注册赠送",
		CreatedAt:    now,
	}
	if err := tx.Create(&creditTx).Error; err != nil {
		tx.Rollback()
		return db.User{}, fmt.Errorf("create credit tx failed: %w", err)
	}

	// Create welcome notification
	notification := db.UserNotification{
		UserID:    user.ID,
		BizKey:    fmt.Sprintf("welcome_%d", user.ID),
		Title:     "欢迎来到 小野 AI",
		Summary:   "注册成功！你已获得 10 钻石，每日签到可免费领取更多钻石。",
		Content:   "Hi，欢迎加入 小野 AI！\n\n你已获得 10 颗钻石作为新人礼物，可以立即开始 AI 创作。\n\n每日签到即可领取更多钻石，连续签到奖励更多！",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&notification).Error; err != nil {
		tx.Rollback()
		return db.User{}, fmt.Errorf("create notification failed: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return db.User{}, fmt.Errorf("commit failed: %w", err)
	}

	return user, nil
}
