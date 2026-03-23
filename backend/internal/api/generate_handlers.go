package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/provider"
	"google-ai-proxy/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// updateGenerationFailed 更新生成记录为失败状态并退还钻石
func updateGenerationFailed(genID uint64, errMsg string, credits int, userID uint64) {
	db.DB.Model(&db.Generation{}).Where("id = ?", genID).Updates(map[string]interface{}{
		"status":     "failed",
		"error_msg":  errMsg,
		"updated_at": time.Now(),
	})
	refundCredits(userID, credits, "generate-failed")
}

// downloadImageAsBase64 从 URL 下载图片并转为 base64
func downloadImageAsBase64(imageURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片失败: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// UnifiedGenerate 统一生成接口（支持 image/video/ecommerce）
// POST /api/generate
func UnifiedGenerate(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var req UnifiedGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 根据类型路由到不同处理
	switch req.Type {
	case "image":
		handleUnifiedImageGenerate(c, userID, req)
	case "video":
		handleUnifiedVideoGenerate(c, userID, req)
	case "ecommerce":
		handleUnifiedEcommerceGenerate(c, userID, req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的生成类型: " + req.Type})
	}
}

// handleUnifiedImageGenerate 统一接口 - 图片生成
func handleUnifiedImageGenerate(c *gin.Context, userID uint64, req UnifiedGenerateRequest) {
	startTime := time.Now()
	userIDStr := strconv.FormatUint(userID, 10)

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	// 验证图片数量
	if len(req.Images) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多上传 3 张图片"})
		return
	}

	// 从 params 中提取参数
	aspectRatio, _ := req.Params["aspectRatio"].(string)
	imageSize, _ := req.Params["imageSize"].(string)
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if imageSize == "" {
		imageSize = "2K"
	}

	// 获取模型，默认使用 Nanobanana
	model := req.Model
	if model == "" {
		model = ModelNanobanana
	}

	mode := "text-to-image"
	if len(req.Images) > 0 {
		mode = "image-editing"
	}

	requiredCredits := GetImageCredits(model, imageSize)

	if user.Credits < requiredCredits {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":            "钻石不足",
			"required_credits": requiredCredits,
			"current_balance":  user.Credits,
		})
		return
	}

	// 扣除钻石
	deductResult := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, requiredCredits).
		Updates(map[string]interface{}{
			"credits":     gorm.Expr("credits - ?", requiredCredits),
			"usage_count": gorm.Expr("usage_count + ?", 1),
			"updated_at":  time.Now(),
		})
	if deductResult.Error != nil || deductResult.RowsAffected == 0 {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足或扣费失败"})
		return
	}
	if err := recordCreditTransaction(
		db.DB,
		userID,
		-requiredCredits,
		CreditTxTypeGenerateCost,
		"generate",
		"",
		"image generation",
	); err != nil {
		refundCredits(userID, requiredCredits, "ledger-write-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录钻石流水失败"})
		return
	}

	// 创建生成记录
	params := map[string]interface{}{
		"model":       model,
		"mode":        mode,
		"aspectRatio": aspectRatio,
		"imageSize":   imageSize,
	}
	genReq := CreateGenerationRequest{
		Type:            "image",
		Prompt:          req.Prompt,
		ReferenceImages: req.Images,
		Params:          params,
		Images:          []string{},
		Status:          "generating",
		CreditsCost:     requiredCredits,
	}
	genRecord, err := CreateGeneration(userID, genReq)
	if err != nil {
		refundCredits(userID, requiredCredits, "create-record-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建生成记录失败"})
		return
	}
	genID := genRecord.ID

	// 后台生成
	go func() {
		inputBase64s := make([]string, 0, len(req.Images))
		for i, imgURL := range req.Images {
			base64Data, err := downloadImageAsBase64(imgURL)
			if err != nil {
				log.Printf("下载输入图像 %d 失败 [用户:%d]: %v", i+1, userID, err)
				updateGenerationFailed(genID, "获取输入图像失败", requiredCredits, userID)
				return
			}
			inputBase64s = append(inputBase64s, base64Data)
		}

		// 下载 mask 图片（局部重绘）
		var maskBase64 string
		if req.Mask != "" {
			maskBase64, err = downloadImageAsBase64(req.Mask)
			if err != nil {
				log.Printf("下载 mask 图片失败 [用户:%d]: %v", userID, err)
				updateGenerationFailed(genID, "获取 mask 图片失败", requiredCredits, userID)
				return
			}
		}

		// 统一使用归一化后的 model 变量，确保空值/非法值时也能正确路由
		generator, genErr := provider.Get(model)
		if genErr != nil {
			updateGenerationFailed(genID, "模型加载失败: "+genErr.Error(), requiredCredits, userID)
			return
		}

		result, genErr := generator.GenerateImage(req.Prompt, provider.ImageOptions{
			AspectRatio: aspectRatio,
			ImageSize:   imageSize,
			InputImages: inputBase64s,
			MaskImage:   maskBase64,
		})
		if genErr != nil {
			updateGenerationFailed(genID, "生成失败: "+genErr.Error(), requiredCredits, userID)
			return
		}

		outputImageURL, err := storage.UploadBase64Image(result.Data, userIDStr, "banana")
		if err != nil {
			updateGenerationFailed(genID, "上传结果失败", requiredCredits, userID)
			return
		}

		imagesJSON, _ := json.Marshal([]string{outputImageURL})
		db.DB.Model(&db.Generation{}).Where("id = ?", genID).Updates(map[string]interface{}{
			"images":     string(imagesJSON),
			"status":     "success",
			"updated_at": time.Now(),
		})

		log.Printf("图像生成成功 [用户:%d] [记录:%d]", userID, genID)
	}()

	log.Printf("图像生成任务创建 [用户:%d] [记录:%d]", userID, genID)

	resp := gin.H{
		"task_id":           genID,
		"status":            "generating",
		"credits_spent":     requiredCredits,
		"credits_remaining": user.Credits - requiredCredits,
	}

	logAPICall("/api/generate", req, http.StatusOK, resp, time.Since(startTime), userIDStr)
	c.JSON(http.StatusAccepted, resp)
}

// handleUnifiedVideoGenerate 统一接口 - 视频生成
func handleUnifiedVideoGenerate(c *gin.Context, userID uint64, req UnifiedGenerateRequest) {
	startTime := time.Now()

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	// 从 params 提取视频参数
	mode, _ := req.Params["mode"].(string)
	resolution, _ := req.Params["resolution"].(string)
	ratio, _ := req.Params["ratio"].(string)
	duration := 5
	if d, ok := req.Params["duration"].(float64); ok {
		duration = int(d)
	}
	generateAudio := false
	if ga, ok := req.Params["generate_audio"].(bool); ok {
		generateAudio = ga
	}
	firstFrame, _ := req.Params["first_frame"].(string)
	lastFrame, _ := req.Params["last_frame"].(string)

	// 默认值
	if mode == "" {
		mode = "text-to-video"
	}
	if resolution == "" {
		resolution = "720p"
	}
	if ratio == "" {
		ratio = "16:9"
	}

	model := req.Model
	if model == "" {
		model = "doubao-seedance-1-5-pro-251215"
	}

	videoProvider := provider.GetVideoProviderForModel(model)
	if videoProvider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频模型: " + model})
		return
	}

	if !videoProvider.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "视频生成服务暂不可用"})
		return
	}

	requiredCredits := videoProvider.CalculateCredits(resolution, duration, generateAudio)

	if user.Credits < requiredCredits {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":            "钻石不足",
			"required_credits": requiredCredits,
			"current_balance":  user.Credits,
		})
		return
	}

	// 扣除钻石
	deductResult := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, requiredCredits).
		Updates(map[string]interface{}{
			"credits":    gorm.Expr("credits - ?", requiredCredits),
			"updated_at": time.Now(),
		})
	if deductResult.Error != nil || deductResult.RowsAffected == 0 {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足或扣费失败"})
		return
	}
	if err := recordCreditTransaction(
		db.DB,
		userID,
		-requiredCredits,
		CreditTxTypeGenerateCost,
		"generate",
		"",
		"video generation",
	); err != nil {
		refundCredits(userID, requiredCredits, "ledger-write-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录钻石流水失败"})
		return
	}

	// 下载首帧/尾帧
	var firstFrameBase64, lastFrameBase64 string
	var err error
	if firstFrame != "" {
		firstFrameBase64, err = downloadImageAsBase64(firstFrame)
		if err != nil {
			refundCredits(userID, requiredCredits, "download-first-frame-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取首帧图片失败"})
			return
		}
	}
	if lastFrame != "" {
		lastFrameBase64, err = downloadImageAsBase64(lastFrame)
		if err != nil {
			refundCredits(userID, requiredCredits, "download-last-frame-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取尾帧图片失败"})
			return
		}
	}

	// 下载参考图（Veo 3.1 支持最多 3 张）
	var refImageBase64s []string
	if len(req.Images) > 0 {
		for i, imgURL := range req.Images {
			if i >= 3 {
				break
			}
			base64Data, err := downloadImageAsBase64(imgURL)
			if err != nil {
				log.Printf("[Video] 下载参考图 %d 失败: %v", i+1, err)
				continue
			}
			refImageBase64s = append(refImageBase64s, base64Data)
		}
	}

	result, err := videoProvider.CreateVideoTask(provider.VideoGenerateRequest{
		Model:           model,
		Prompt:          req.Prompt,
		Mode:            mode,
		Resolution:      resolution,
		Ratio:           ratio,
		Duration:        duration,
		GenerateAudio:   generateAudio,
		FirstFrame:      firstFrameBase64,
		LastFrame:       lastFrameBase64,
		ReferenceImages: refImageBase64s,
	})

	if err != nil {
		refundCredits(userID, requiredCredits, "provider-create-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 Generation 记录
	refImages := []string{}
	if firstFrame != "" {
		refImages = append(refImages, firstFrame)
	}
	if lastFrame != "" {
		refImages = append(refImages, lastFrame)
	}
	refImages = append(refImages, req.Images...)
	params := map[string]interface{}{
		"model":         model,
		"provider":      videoProvider.GetProviderName(),
		"mode":          mode,
		"resolution":    resolution,
		"ratio":         ratio,
		"duration":      duration,
		"generateAudio": generateAudio,
	}
	genReq := CreateGenerationRequest{
		Type:            "video",
		Prompt:          req.Prompt,
		ReferenceImages: refImages,
		Params:          params,
		Status:          "queued",
		CreditsCost:     requiredCredits,
		TaskID:          &result.TaskID,
	}
	genRecord, err := CreateGeneration(userID, genReq)
	if err != nil {
		refundCredits(userID, requiredCredits, "video-save-record-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务记录失败"})
		return
	}

	db.DB.First(user, userID)

	log.Printf("[Video] 任务创建成功 [用户:%d] [内部ID:%d] [服务商ID:%s]", userID, genRecord.ID, result.TaskID)

	resp := gin.H{
		"task_id":           genRecord.ID,
		"provider_task_id":  result.TaskID,
		"status":            "queued",
		"credits_spent":     requiredCredits,
		"credits_remaining": user.Credits,
	}

	userIDStr := strconv.FormatUint(userID, 10)
	logAPICall("/api/generate", req, http.StatusOK, resp, time.Since(startTime), userIDStr)
	c.JSON(http.StatusOK, resp)
}

// handleUnifiedEcommerceGenerate 统一接口 - 商品组图
func handleUnifiedEcommerceGenerate(c *gin.Context, userID uint64, req UnifiedGenerateRequest) {
	startTime := time.Now()
	userIDStr := strconv.FormatUint(userID, 10)

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	// 验证图片
	if len(req.Images) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少上传1张参考图片"})
		return
	}
	if len(req.Images) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多上传3张参考图片"})
		return
	}

	// 从 params 提取参数
	outputCount := 7
	if oc, ok := req.Params["outputCount"].(float64); ok {
		outputCount = int(oc)
	}
	if outputCount < 5 || outputCount > 15 {
		outputCount = 7
	}
	aspectRatio, _ := req.Params["aspectRatio"].(string)
	imageSize, _ := req.Params["imageSize"].(string)
	imageType, _ := req.Params["imageType"].(string)
	ecommerceType, _ := req.Params["ecommerceType"].(string)

	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if imageSize == "" {
		imageSize = "2K"
	}

	promptSuffix := GenerateEcommercePromptSuffix(imageType, ecommerceType, outputCount)
	fullPrompt := req.Prompt + promptSuffix

	requiredCredits := GetEcommerceCredits(imageSize, outputCount)

	if user.Credits < requiredCredits {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":            "钻石不足",
			"required_credits": requiredCredits,
			"current_balance":  user.Credits,
		})
		return
	}

	// 扣除钻石
	deductResult := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, requiredCredits).
		Updates(map[string]interface{}{
			"credits":     gorm.Expr("credits - ?", requiredCredits),
			"usage_count": gorm.Expr("usage_count + ?", 1),
			"updated_at":  time.Now(),
		})
	if deductResult.Error != nil || deductResult.RowsAffected == 0 {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足或扣费失败"})
		return
	}
	if err := recordCreditTransaction(
		db.DB,
		userID,
		-requiredCredits,
		CreditTxTypeGenerateCost,
		"generate",
		"",
		"ecommerce generation",
	); err != nil {
		refundCredits(userID, requiredCredits, "ledger-write-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录钻石流水失败"})
		return
	}

	modelID := req.Model
	if modelID == "" {
		modelID = DefaultEcommerceModel
	}

	// 上传输入图片
	inputURLs := make([]string, 0, len(req.Images))
	for i, img := range req.Images {
		url, err := storage.UploadBase64Image(img, userIDStr, "ecommerce-input")
		if err != nil {
			log.Printf("[电商中心] 上传输入图片 %d 失败: %v", i+1, err)
			refundCredits(userID, requiredCredits, "ecommerce-upload-input-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "上传图片失败"})
			return
		}
		inputURLs = append(inputURLs, url)
	}

	// 创建生成记录
	params := map[string]interface{}{
		"model":         modelID,
		"mode":          "multi-image-group",
		"aspectRatio":   aspectRatio,
		"imageSize":     imageSize,
		"outputCount":   outputCount,
		"imageType":     imageType,
		"ecommerceType": ecommerceType,
	}
	genReq := CreateGenerationRequest{
		Type:            "image",
		Prompt:          fullPrompt,
		ReferenceImages: inputURLs,
		Params:          params,
		Images:          []string{},
		Status:          "generating",
		CreditsCost:     requiredCredits,
	}
	genRecord, err := CreateGeneration(userID, genReq)
	if err != nil {
		refundCredits(userID, requiredCredits, "ecommerce-create-record-failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建生成记录失败"})
		return
	}
	genID := genRecord.ID

	// 后台生成
	go func() {
		generator, err := provider.Get(modelID)
		if err != nil {
			updateGenerationFailed(genID, "模型加载失败", requiredCredits, userID)
			return
		}

		multiGen, ok := generator.(provider.MultiImageGenerator)
		if !ok || !multiGen.SupportsMultiImage() {
			updateGenerationFailed(genID, "所选模型不支持多图生组图", requiredCredits, userID)
			return
		}

		result, genErr := multiGen.GenerateMultiImage(fullPrompt, req.Images, outputCount, provider.ImageOptions{
			AspectRatio: aspectRatio,
			ImageSize:   imageSize,
		})
		if genErr != nil {
			updateGenerationFailed(genID, "生成失败: "+genErr.Error(), requiredCredits, userID)
			return
		}

		outputURLs := make([]string, 0, len(result.Images))
		for i, img := range result.Images {
			url, err := storage.UploadBase64Image(img.Data, userIDStr, "ecommerce-output")
			if err != nil {
				log.Printf("[电商中心] 上传输出图片 %d 失败: %v", i+1, err)
				continue
			}
			outputURLs = append(outputURLs, url)
		}

		if len(outputURLs) == 0 {
			updateGenerationFailed(genID, "所有图片上传失败", requiredCredits, userID)
			return
		}

		actualCreditsSpent := GetEcommerceCredits(imageSize, len(outputURLs))
		if refundAmount := requiredCredits - actualCreditsSpent; refundAmount > 0 {
			refundCredits(userID, refundAmount, "ecommerce-partial-refund")
		}

		outputImagesJSON, _ := json.Marshal(outputURLs)
		db.DB.Model(&db.Generation{}).Where("id = ?", genID).Updates(map[string]interface{}{
			"images":       string(outputImagesJSON),
			"credits_cost": actualCreditsSpent,
			"status":       "success",
			"updated_at":   time.Now(),
		})

		log.Printf("[电商中心] 生成成功 [用户:%d] [记录:%d]: 输出%d张", userID, genID, len(outputURLs))
	}()

	log.Printf("[电商中心] 生成任务创建 [用户:%d] [记录:%d]", userID, genID)

	resp := gin.H{
		"task_id":           genID,
		"status":            "generating",
		"credits_spent":     requiredCredits,
		"credits_remaining": user.Credits - requiredCredits,
	}

	logAPICall("/api/generate", req, http.StatusAccepted, resp, time.Since(startTime), userIDStr)
	c.JSON(http.StatusAccepted, resp)
}
