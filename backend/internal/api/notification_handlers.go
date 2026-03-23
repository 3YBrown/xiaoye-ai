package api

import (
	"net/http"
	"strconv"
	"time"

	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
)

func parseNotificationPagination(c *gin.Context) (limit int, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return
}

// ListUserNotifications lists current user's notifications.
func ListUserNotifications(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	limit, offset := parseNotificationPagination(c)
	query := db.DB.Model(&db.UserNotification{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query notifications"})
		return
	}

	var unreadCount int64
	if err := db.DB.Model(&db.UserNotification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query notifications"})
		return
	}

	var rows []db.UserNotification
	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query notifications"})
		return
	}

	items := make([]gin.H, len(rows))
	for i, row := range rows {
		items[i] = gin.H{
			"id":         row.ID,
			"title":      row.Title,
			"summary":    row.Summary,
			"content":    row.Content,
			"is_read":    row.IsRead,
			"created_at": row.CreatedAt.UnixMilli(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":        items,
		"total":        total,
		"unread_count": unreadCount,
		"limit":        limit,
		"offset":       offset,
	})
}

// MarkNotificationRead marks one notification as read for current user.
func MarkNotificationRead(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || notificationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	result := db.DB.Model(&db.UserNotification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// MarkAllNotificationsRead marks all unread notifications as read for current user.
func MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	result := db.DB.Model(&db.UserNotification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"updated": result.RowsAffected,
	})
}
