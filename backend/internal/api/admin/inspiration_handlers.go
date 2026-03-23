package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var allowedReviewStatus = map[string]struct{}{
	"pending":  {},
	"approved": {},
	"rejected": {},
}

const (
	creditTxTypeInspirationReviewReward = "inspiration_review_reward"
	creditTxSourceInspirationReview     = "inspiration_review"
)

func getReviewRewardByPostType(postType string) int {
	if strings.ToLower(strings.TrimSpace(postType)) == "video" {
		return 4
	}
	return 2
}

func appendReviewRewardAndNotification(tx *gorm.DB, post db.InspirationPost) error {
	reward := getReviewRewardByPostType(post.Type)
	if reward <= 0 {
		return nil
	}

	sourceID := strconv.FormatUint(post.ID, 10)
	var existingCount int64
	if err := tx.Model(&db.CreditTransaction{}).
		Where("user_id = ? AND type = ? AND source = ? AND source_id = ?",
			post.UserID, creditTxTypeInspirationReviewReward, creditTxSourceInspirationReview, sourceID).
		Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 {
		return nil
	}

	if err := tx.Model(&db.User{}).
		Where("id = ?", post.UserID).
		Updates(map[string]interface{}{
			"credits":    gorm.Expr("credits + ?", reward),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return err
	}

	var user db.User
	if err := tx.Select("id", "credits").First(&user, post.UserID).Error; err != nil {
		return err
	}

	txRecord := db.CreditTransaction{
		UserID:       post.UserID,
		Delta:        reward,
		BalanceAfter: user.Credits,
		Type:         creditTxTypeInspirationReviewReward,
		Source:       creditTxSourceInspirationReview,
		SourceID:     sourceID,
		Note:         fmt.Sprintf("inspiration post %d approved reward", post.ID),
		CreatedAt:    time.Now(),
	}
	if err := tx.Create(&txRecord).Error; err != nil {
		return err
	}

	notification := db.UserNotification{
		UserID:    post.UserID,
		BizKey:    "inspiration-review-approved-" + sourceID,
		Title:     "灵感内容审核通过",
		Summary:   fmt.Sprintf("你的内容已通过审核，已获得 %d 钻石奖励。", reward),
		Content:   fmt.Sprintf("恭喜！你发布的灵感内容（ID:%d）审核通过，系统已发放 %d 钻石。", post.ID, reward),
		IsRead:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "biz_key"},
		},
		DoNothing: true,
	}).Create(&notification).Error
}

func parseDateOnly(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Millisecond)
	}
	return &parsed, nil
}

func parseListPagination(c *gin.Context) (limit int, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return
}

func parseJSONStringArray(raw string) []string {
	if raw == "" || raw == "[]" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func buildInspirationResponse(post db.InspirationPost, author db.User) inspirationResponse {
	publishedAt := post.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = post.CreatedAt
	}

	mediaURLs := parseJSONStringArray(post.MediaURLs)
	images := []string{}
	videoURL := ""
	postType := strings.ToLower(strings.TrimSpace(post.Type))
	if postType == "video" {
		if len(mediaURLs) > 0 {
			videoURL = mediaURLs[0]
		}
	} else {
		images = mediaURLs
	}

	coverURL := strings.TrimSpace(post.CoverURL)
	if coverURL == "" {
		if len(images) > 0 {
			coverURL = images[0]
		} else {
			coverURL = videoURL
		}
	}

	reviewedAtUnix := int64(0)
	if post.ReviewedAt != nil && !post.ReviewedAt.IsZero() {
		reviewedAtUnix = post.ReviewedAt.UnixMilli()
	}

	return inspirationResponse{
		ID:               post.ID,
		ShareID:          post.ShareID,
		Type:             post.Type,
		Title:            post.Title,
		Prompt:           post.Prompt,
		Images:           images,
		VideoURL:         videoURL,
		CoverURL:         coverURL,
		ReviewStatus:     post.ReviewStatus,
		ReviewedBySource: post.ReviewedBySource,
		ReviewedByID:     post.ReviewedByID,
		ReviewedAt:       reviewedAtUnix,
		PublishedAt:      publishedAt.UnixMilli(),
		Author: authorResponse{
			UserID:   author.ID,
			Nickname: author.Nickname,
			Avatar:   author.Avatar,
		},
	}
}

// ListInspirations lists inspiration posts for moderation and audit.
func ListInspirations(c *gin.Context) {
	limit, offset := parseListPagination(c)

	reviewStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("review_status", "pending")))
	if reviewStatus == "" {
		reviewStatus = "pending"
	}
	if reviewStatus != "all" {
		if _, ok := allowedReviewStatus[reviewStatus]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review_status"})
			return
		}
	}

	var userID uint64
	userIDRaw := strings.TrimSpace(c.Query("user_id"))
	if userIDRaw != "" {
		parsedUserID, err := strconv.ParseUint(userIDRaw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userID = parsedUserID
	}

	startAt, err := parseDateOnly(c.Query("start_date"), false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date, expected YYYY-MM-DD"})
		return
	}
	endAt, err := parseDateOnly(c.Query("end_date"), true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date, expected YYYY-MM-DD"})
		return
	}

	keyword := strings.TrimSpace(c.Query("q"))

	query := db.DB.Model(&db.InspirationPost{}).Where("inspiration_posts.status = ?", "published")
	if reviewStatus != "all" {
		query = query.Where("inspiration_posts.review_status = ?", reviewStatus)
	}
	if userID > 0 {
		query = query.Where("inspiration_posts.user_id = ?", userID)
	}
	if startAt != nil {
		query = query.Where("inspiration_posts.published_at >= ?", *startAt)
	}
	if endAt != nil {
		query = query.Where("inspiration_posts.published_at <= ?", *endAt)
	}
	if keyword != "" {
		likeExpr := "%" + keyword + "%"
		query = query.Where(
			"(inspiration_posts.share_id LIKE ? OR inspiration_posts.title LIKE ? OR inspiration_posts.prompt LIKE ?)",
			likeExpr, likeExpr, likeExpr,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	var posts []db.InspirationPost
	if err := query.
		Order("FIELD(inspiration_posts.review_status, 'pending', 'rejected', 'approved')").
		Order("inspiration_posts.published_at DESC").
		Order("inspiration_posts.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	userIDSet := map[uint64]struct{}{}
	for _, post := range posts {
		userIDSet[post.UserID] = struct{}{}
	}
	userIDs := make([]uint64, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	authors := map[uint64]db.User{}
	if len(userIDs) > 0 {
		var users []db.User
		if err := db.DB.Select("id", "nickname", "avatar").Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, user := range users {
				authors[user.ID] = user
			}
		}
	}

	items := make([]inspirationResponse, 0, len(posts))
	for _, post := range posts {
		author := authors[post.UserID]
		if author.ID == 0 {
			author = db.User{ID: post.UserID, Nickname: "Creator"}
		}
		items = append(items, buildInspirationResponse(post, author))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
		"review_status": reviewStatus,
	})
}

// ReviewInspiration updates review status for an inspiration post.
func ReviewInspiration(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req reviewInspirationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	targetStatus := ""
	switch action {
	case "approve":
		targetStatus = "approved"
	case "reject":
		targetStatus = "rejected"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}

	note := strings.TrimSpace(req.Note)
	operatorSource := strings.TrimSpace(c.GetString("adminOperatorSource"))
	if operatorSource == "" {
		operatorSource = "admin_console"
	}
	operatorID := strings.TrimSpace(c.GetString("adminOperatorID"))
	if operatorID == "" {
		operatorID = "unknown"
	}

	var updatedPost db.InspirationPost
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var post db.InspirationPost
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, postID).Error; err != nil {
			return err
		}

		fromStatus := strings.TrimSpace(post.ReviewStatus)
		if fromStatus != targetStatus {
			now := time.Now()
			if err := tx.Model(&db.InspirationPost{}).
				Where("id = ?", postID).
				Updates(map[string]interface{}{
					"review_status":      targetStatus,
					"reviewed_by_source": operatorSource,
					"reviewed_by_id":     operatorID,
					"reviewed_at":        now,
					"updated_at":         now,
				}).Error; err != nil {
				return err
			}

			logRow := db.InspirationReviewLog{
				PostID:         postID,
				Action:         action,
				FromStatus:     fromStatus,
				ToStatus:       targetStatus,
				Note:           note,
				OperatorSource: operatorSource,
				OperatorID:     operatorID,
			}
			if err := tx.Create(&logRow).Error; err != nil {
				return err
			}

			if targetStatus == "approved" && fromStatus != "approved" {
				if err := appendReviewRewardAndNotification(tx, post); err != nil {
					return err
				}
			}
		}

		if err := tx.First(&post, postID).Error; err != nil {
			return err
		}
		updatedPost = post
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to review post"})
		return
	}

	var author db.User
	if err := db.DB.Select("id", "nickname", "avatar").First(&author, updatedPost.UserID).Error; err != nil {
		author = db.User{ID: updatedPost.UserID, Nickname: "Creator"}
	}

	c.JSON(http.StatusOK, gin.H{
		"item": buildInspirationResponse(updatedPost, author),
	})
}
