package api

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/provider"
	"google-ai-proxy/internal/storage"
)

// ============ 后台视频任务状态轮询 Worker ============

var processingTasks sync.Map // 防止同一任务被并发处理

// StartVideoTaskPoller 启动后台视频任务状态轮询（在 main.go 中调用）
func StartVideoTaskPoller() {
	go func() {
		log.Println("[VideoPoller] 后台视频任务状态轮询已启动，间隔: 10秒")
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			pollPendingVideoTasks()
		}
	}()
}

// pollPendingVideoTasks 扫描并处理所有未完成的视频任务
func pollPendingVideoTasks() {
	var generations []db.Generation
	if err := db.DB.Where("type = ? AND status IN ?", "video", []string{"queued", "running", "pending"}).
		Find(&generations).Error; err != nil {
		log.Printf("[VideoPoller] 查询待处理任务失败: %v", err)
		return
	}
	if len(generations) == 0 {
		return
	}
	log.Printf("[VideoPoller] 发现 %d 个待处理任务", len(generations))
	for i := range generations {
		go processVideoGeneration(&generations[i])
	}
}

// processVideoGeneration 处理单个视频生成记录：查询状态、下载视频、更新DB、退还钻石
func processVideoGeneration(gen *db.Generation) {
	if gen.TaskID == nil || *gen.TaskID == "" {
		return
	}
	taskID := *gen.TaskID

	// 防止同一任务并发处理
	if _, loaded := processingTasks.LoadOrStore(taskID, true); loaded {
		return
	}
	defer processingTasks.Delete(taskID)

	// 再次从DB读取最新状态
	var fresh db.Generation
	if err := db.DB.Where("task_id = ?", taskID).First(&fresh).Error; err != nil {
		log.Printf("[VideoPoller] 重新查询任务失败 [task:%s]: %v", taskID, err)
		return
	}
	if fresh.Status == "success" || fresh.Status == "failed" || fresh.Status == "expired" {
		return // 已终态，跳过
	}
	gen = &fresh

	// 超时检测：超过30分钟视为卡死，标记失败并退还钻石
	if time.Since(gen.CreatedAt) > 30*time.Minute {
		log.Printf("[VideoPoller] 任务超时 [task:%s]，标记为失败并退还钻石", taskID)
		updates := map[string]interface{}{
			"status":     "failed",
			"error_msg":  "任务处理超时，已自动退还钻石",
			"updated_at": time.Now(),
		}
		if err := db.DB.Model(gen).Updates(updates).Error; err == nil {
			refundCredits(gen.UserID, gen.CreditsCost, "video-task-"+taskID)
		}
		return
	}

	// 从 params 中获取 provider 信息
	providerName := "volcengine" // 默认
	if gen.Params != "" {
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(gen.Params), &params); err == nil {
			if p, ok := params["provider"].(string); ok && p != "" {
				providerName = p
			}
		}
	}

	// 获取对应的 Provider
	videoProvider := provider.GetVideoProvider(providerName)
	if videoProvider == nil {
		videoProvider = provider.GetVideoProvider("volcengine") // 兼容旧数据
	}
	if videoProvider == nil {
		log.Printf("[VideoPoller] 未找到 Provider [task:%s, provider:%s]", taskID, providerName)
		return
	}

	// 向服务商查询最新状态
	statusResult, err := videoProvider.GetVideoTaskStatus(taskID)
	if err != nil {
		log.Printf("[VideoPoller] 查询任务状态失败 [task:%s]: %v", taskID, err)
		return
	}

	updates := make(map[string]interface{})
	needUpdate := false

	// 状态映射：服务商 succeeded → generations success
	newStatus := statusResult.Status
	if newStatus == "succeeded" {
		newStatus = "success"
	}

	if newStatus != gen.Status {
		updates["status"] = newStatus
		needUpdate = true
		log.Printf("[VideoPoller] 任务状态变更 [task:%s]: %s -> %s", taskID, gen.Status, newStatus)
	}

	// 成功：下载视频并转存到 OSS
	if statusResult.Status == "succeeded" && statusResult.VideoURL != "" {
		// Google Veo 下载需要 API key
		var downloadHeaders map[string]string
		if providerName == "google" {
			downloadHeaders = map[string]string{
				"x-goog-api-key": config.GetGoogleAPIKey(),
			}
		}
		videoURL, err := storage.DownloadAndUploadVideo(statusResult.VideoURL, strconv.FormatUint(gen.UserID, 10), downloadHeaders)
		if err != nil {
			log.Printf("[VideoPoller] 转存视频失败 [task:%s]: %v，使用原始URL", taskID, err)
			updates["video_url"] = statusResult.VideoURL
		} else {
			updates["video_url"] = videoURL
			log.Printf("[VideoPoller] 视频转存成功 [task:%s]", taskID)
		}
		if statusResult.CoverURL != "" {
			params := map[string]interface{}{}
			if gen.Params != "" && gen.Params != "{}" {
				_ = json.Unmarshal([]byte(gen.Params), &params)
			}
			if params == nil {
				params = map[string]interface{}{}
			}
			params["coverUrl"] = statusResult.CoverURL
			if b, err := json.Marshal(params); err == nil {
				updates["params"] = string(b)
				gen.Params = string(b)
			}
		}
		needUpdate = true
	}

	// 失败：记录错误信息
	if statusResult.Status == "failed" {
		if statusResult.ErrorMessage != "" {
			updates["error_msg"] = statusResult.ErrorMessage
		}
		needUpdate = true
	}

	if needUpdate {
		updates["updated_at"] = time.Now()
		if err := db.DB.Model(gen).Updates(updates).Error; err != nil {
			log.Printf("[VideoPoller] 更新任务状态失败 [task:%s]: %v", taskID, err)
			return
		}
		// DB 更新成功后再退还钻石（防止重复退款）
		if statusResult.Status == "failed" {
			refundCredits(gen.UserID, gen.CreditsCost, "video-task-"+taskID)
		}
	}
}
