package api

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"google-ai-proxy/internal/storage"
)

// UploadImage 上传图片到 OSS，返回 URL
func UploadImage(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var req UploadImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	userIDStr := strconv.FormatUint(userID, 10)
	url, err := storage.UploadBase64Image(req.Image, userIDStr, "useredit")
	if err != nil {
		log.Printf("上传图片失败 [用户:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传图片失败"})
		return
	}

	c.JSON(http.StatusOK, UploadImageResponse{URL: url})
}

// UploadVideo uploads a user provided video file to OSS and returns public URL.
func UploadVideo(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	if file.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty file"})
		return
	}
	if file.Size > 100*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file too large"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".mp4", ".mov", ".webm", ".m4v":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported video format"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}
	defer src.Close()

	videoData, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read video data"})
		return
	}

	userIDStr := strconv.FormatUint(userID, 10)
	url, err := storage.UploadVideoData(videoData, userIDStr, ext)
	if err != nil {
		log.Printf("upload video failed [user:%d]: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload video"})
		return
	}

	c.JSON(http.StatusOK, UploadImageResponse{URL: url})
}
