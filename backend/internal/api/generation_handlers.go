package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type generationShareMeta struct {
	SourceGenerationID uint64 `gorm:"column:source_generation_id"`
	ShareID            string `gorm:"column:share_id"`
}

func parseBoolQuery(c *gin.Context, key string) bool {
	return c.DefaultQuery(key, "false") == "true"
}

func getShareInfoMap(userID uint64, generationIDs []uint64) (map[uint64]string, error) {
	if len(generationIDs) == 0 {
		return map[uint64]string{}, nil
	}

	var rows []generationShareMeta
	err := db.DB.Model(&db.InspirationPost{}).
		Select("source_generation_id", "share_id").
		Where("user_id = ? AND status = ? AND source_generation_id IN ?", userID, "published", generationIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]string, len(rows))
	for _, row := range rows {
		result[row.SourceGenerationID] = row.ShareID
	}
	return result, nil
}

func getGenerationShareID(userID, generationID uint64) (string, error) {
	var row generationShareMeta
	err := db.DB.Model(&db.InspirationPost{}).
		Select("source_generation_id", "share_id").
		Where("user_id = ? AND status = ? AND source_generation_id = ?", userID, "published", generationID).
		Limit(1).Find(&row).Error
	if err != nil {
		return "", err
	}
	return row.ShareID, nil
}

// ListGenerations returns private generation history for the current user.
func ListGenerations(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	genType := c.Query("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	favoriteOnly := parseBoolQuery(c, "favorite")
	sharedOnly := parseBoolQuery(c, "shared")

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	query := db.DB.Model(&db.Generation{}).Where("generations.user_id = ?", userID)
	if genType != "" && genType != "all" {
		query = query.Where("generations.type = ?", genType)
	}
	if favoriteOnly {
		query = query.Where("generations.is_favorite = ?", true)
	}
	if sharedOnly {
		query = query.Joins(
			"JOIN inspiration_posts ON inspiration_posts.source_generation_id = generations.id AND inspiration_posts.user_id = ? AND inspiration_posts.status = ?",
			userID,
			"published",
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query generations"})
		return
	}

	var generations []db.Generation
	if err := query.Order("generations.created_at DESC").Limit(limit).Offset(offset).Find(&generations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query generations"})
		return
	}

	generationIDs := make([]uint64, 0, len(generations))
	for _, gen := range generations {
		generationIDs = append(generationIDs, gen.ID)
	}

	shareInfo, err := getShareInfoMap(userID, generationIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query share status"})
		return
	}

	items := make([]GenerationResponse, len(generations))
	for i, gen := range generations {
		items[i] = dbGenerationToResponse(gen, shareInfo[gen.ID])
	}

	c.JSON(http.StatusOK, gin.H{
		"generations": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetGeneration returns one private generation record for the current user.
func GetGeneration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	genID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation id"})
		return
	}

	var gen db.Generation
	if err := db.DB.Where("id = ? AND user_id = ?", genID, userID).First(&gen).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query generation"})
		return
	}

	shareID, err := getGenerationShareID(userID, gen.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query share status"})
		return
	}

	c.JSON(http.StatusOK, dbGenerationToResponse(gen, shareID))
}

// CreateGeneration creates a generation row used by /api/generate async workflow.
func CreateGeneration(userID uint64, req CreateGenerationRequest) (*db.Generation, error) {
	refImagesJSON := "[]"
	if len(req.ReferenceImages) > 0 {
		if b, err := json.Marshal(req.ReferenceImages); err == nil {
			refImagesJSON = string(b)
		}
	}

	paramsJSON := "{}"
	if req.Params != nil {
		if b, err := json.Marshal(req.Params); err == nil {
			paramsJSON = string(b)
		}
	}

	imagesJSON := "[]"
	if len(req.Images) > 0 {
		if b, err := json.Marshal(req.Images); err == nil {
			imagesJSON = string(b)
		}
	}

	if req.Status == "" {
		req.Status = "generating"
	}

	gen := db.Generation{
		UserID:          userID,
		Type:            req.Type,
		Prompt:          req.Prompt,
		ReferenceImages: refImagesJSON,
		Params:          paramsJSON,
		Images:          imagesJSON,
		VideoURL:        req.VideoURL,
		Status:          req.Status,
		CreditsCost:     req.CreditsCost,
		ErrorMsg:        req.ErrorMsg,
		TaskID:          req.TaskID,
	}

	if err := db.DB.Create(&gen).Error; err != nil {
		return nil, err
	}

	return &gen, nil
}

// UpdateGeneration updates mutable fields on a generation row.
func UpdateGeneration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	genID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation id"})
		return
	}

	var req UpdateGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updates := map[string]interface{}{}

	if len(req.Images) > 0 {
		if b, err := json.Marshal(req.Images); err == nil {
			updates["images"] = string(b)
		}
	}
	if req.VideoURL != "" {
		updates["video_url"] = req.VideoURL
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.CreditsCost > 0 {
		updates["credits_cost"] = req.CreditsCost
	}
	if req.ErrorMsg != "" {
		updates["error_msg"] = req.ErrorMsg
	}
	if req.TaskID != nil {
		updates["task_id"] = *req.TaskID
	}
	if req.IsFavorite != nil {
		updates["is_favorite"] = *req.IsFavorite
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields provided"})
		return
	}

	result := db.DB.Model(&db.Generation{}).Where("id = ? AND user_id = ?", genID, userID).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update generation"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
		return
	}

	var gen db.Generation
	if err := db.DB.First(&gen, genID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query generation"})
		return
	}

	shareID, err := getGenerationShareID(userID, gen.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query share status"})
		return
	}

	c.JSON(http.StatusOK, dbGenerationToResponse(gen, shareID))
}

// DeleteGeneration deletes one private generation row.
func DeleteGeneration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	genID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation id"})
		return
	}

	result := db.DB.Where("id = ? AND user_id = ?", genID, userID).Delete(&db.Generation{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete generation"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func dbGenerationToResponse(gen db.Generation, shareID string) GenerationResponse {
	var refImages []string
	if gen.ReferenceImages != "" && gen.ReferenceImages != "[]" {
		_ = json.Unmarshal([]byte(gen.ReferenceImages), &refImages)
	}
	if refImages == nil {
		refImages = []string{}
	}

	var params map[string]interface{}
	if gen.Params != "" && gen.Params != "{}" {
		_ = json.Unmarshal([]byte(gen.Params), &params)
	}
	if params == nil {
		params = map[string]interface{}{}
	}

	var images []string
	if gen.Images != "" && gen.Images != "[]" {
		_ = json.Unmarshal([]byte(gen.Images), &images)
	}
	if images == nil {
		images = []string{}
	}

	return GenerationResponse{
		ID:              gen.ID,
		Type:            gen.Type,
		Prompt:          gen.Prompt,
		ReferenceImages: refImages,
		Params:          params,
		Images:          images,
		VideoURL:        gen.VideoURL,
		Status:          gen.Status,
		CreditsCost:     gen.CreditsCost,
		ErrorMsg:        gen.ErrorMsg,
		TaskID:          gen.TaskID,
		IsFavorite:      gen.IsFavorite,
		IsShared:        shareID != "",
		ShareID:         shareID,
		CreatedAt:       gen.CreatedAt.UnixMilli(),
		UpdatedAt:       gen.UpdatedAt.UnixMilli(),
	}
}

// StartGenerationCleanup keeps generations durable and only clears stale operational data.
func StartGenerationCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C

			failedCutoff := time.Now().Add(-30 * 24 * time.Hour)
			result := db.DB.Where("status = ? AND created_at < ?", "failed", failedCutoff).Delete(&db.Generation{})
			if result.RowsAffected > 0 {
				log.Printf("[Generation Cleanup] deleted %d failed generations older than 30 days", result.RowsAffected)
			}

			logCutoff := time.Now().Add(-3 * 24 * time.Hour)
			logResult := db.DB.Where("created_at < ?", logCutoff).Delete(&db.APILog{})
			if logResult.RowsAffected > 0 {
				log.Printf("[API Log Cleanup] deleted %d logs older than 3 days", logResult.RowsAffected)
			}
		}
	}()

	log.Println("[Cleanup Worker] running hourly: keep successful generations, delete failed generations after 30 days, delete API logs after 3 days")
}
