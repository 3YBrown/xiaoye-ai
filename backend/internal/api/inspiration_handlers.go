package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"google-ai-proxy/internal/auth"
	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const shareIDAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

const (
	maxInspirationTagCount  = 5
	maxInspirationTagLength = 24
)

type normalizedTag struct {
	Name string
	Slug string
}

type postTagRow struct {
	PostID uint64 `gorm:"column:post_id"`
	Name   string `gorm:"column:name"`
}

type reviewSnapshot struct {
	Status           string
	ReviewedBySource string
	ReviewedByID     string
	ReviewedAt       *time.Time
}

func isInspirationAutoApprove() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("INSPIRATION_AUTO_APPROVE")))
	if raw == "" {
		return true
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func buildInitialReviewSnapshot(now time.Time) reviewSnapshot {
	if !isInspirationAutoApprove() {
		return reviewSnapshot{Status: "pending"}
	}
	reviewedAt := now
	return reviewSnapshot{
		Status:           "approved",
		ReviewedBySource: "system",
		ReviewedByID:     "auto",
		ReviewedAt:       &reviewedAt,
	}
}

func applyReviewSnapshot(post *db.InspirationPost, snapshot reviewSnapshot) {
	post.ReviewStatus = snapshot.Status
	post.ReviewedBySource = snapshot.ReviewedBySource
	post.ReviewedByID = snapshot.ReviewedByID
	post.ReviewedAt = snapshot.ReviewedAt
}

func appendReviewLog(tx *gorm.DB, postID uint64, action, fromStatus, toStatus, note string, operatorUserID uint64) error {
	action = strings.TrimSpace(action)
	if action == "" {
		return nil
	}
	logRow := db.InspirationReviewLog{
		PostID:         postID,
		Action:         action,
		FromStatus:     strings.TrimSpace(fromStatus),
		ToStatus:       strings.TrimSpace(toStatus),
		Note:           strings.TrimSpace(note),
		OperatorSource: "app_user",
		OperatorID:     strconv.FormatUint(operatorUserID, 10),
	}
	return tx.Create(&logRow).Error
}

func newShareID(length int) (string, error) {
	if length <= 0 {
		length = 12
	}
	buf := make([]byte, length)
	max := byte(len(shareIDAlphabet))
	randBytes := make([]byte, length)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		buf[i] = shareIDAlphabet[randBytes[i]%max]
	}
	return string(buf), nil
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

func parseJSONStringMap(raw string) map[string]interface{} {
	if raw == "" || raw == "{}" {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func toJSONStringArray(values []string) string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	b, err := json.Marshal(filtered)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func toJSONStringMap(values map[string]interface{}) string {
	if values == nil {
		return "{}"
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func normalizeTagName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("empty tag")
	}
	if len([]rune(name)) > maxInspirationTagLength {
		return "", errors.New("tag too long")
	}
	return name, nil
}

func normalizeTagSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		switch r {
		case ' ', '-', '_', '#':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func sanitizeTags(rawTags []string) ([]normalizedTag, error) {
	if len(rawTags) == 0 {
		return []normalizedTag{}, nil
	}
	result := make([]normalizedTag, 0, len(rawTags))
	seen := map[string]struct{}{}
	for _, raw := range rawTags {
		name, err := normalizeTagName(raw)
		if err != nil {
			return nil, err
		}
		slug := normalizeTagSlug(name)
		if slug == "" {
			return nil, errors.New("invalid tag")
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		result = append(result, normalizedTag{Name: name, Slug: slug})
		if len(result) > maxInspirationTagCount {
			return nil, errors.New("too many tags")
		}
	}
	return result, nil
}

func upsertTags(tx *gorm.DB, tags []normalizedTag) ([]db.InspirationTag, error) {
	if len(tags) == 0 {
		return []db.InspirationTag{}, nil
	}

	result := make([]db.InspirationTag, 0, len(tags))
	for _, tag := range tags {
		row := db.InspirationTag{
			Name:   tag.Name,
			Slug:   tag.Slug,
			Status: "active",
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return nil, err
		}

		var current db.InspirationTag
		if err := tx.Where("slug = ?", tag.Slug).First(&current).Error; err != nil {
			return nil, err
		}
		if current.Status == "blocked" {
			return nil, errors.New("tag is blocked")
		}
		result = append(result, current)
	}
	return result, nil
}

func syncPostTags(tx *gorm.DB, postID uint64, tags []db.InspirationTag) error {
	var oldLinks []db.InspirationPostTag
	if err := tx.Select("tag_id").Where("post_id = ?", postID).Find(&oldLinks).Error; err != nil {
		return err
	}

	impacted := map[uint64]struct{}{}
	for _, link := range oldLinks {
		impacted[link.TagID] = struct{}{}
	}

	if err := tx.Where("post_id = ?", postID).Delete(&db.InspirationPostTag{}).Error; err != nil {
		return err
	}

	for _, tag := range tags {
		impacted[tag.ID] = struct{}{}
		link := db.InspirationPostTag{
			PostID: postID,
			TagID:  tag.ID,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
			return err
		}
	}

	for tagID := range impacted {
		if err := tx.Model(&db.InspirationTag{}).
			Where("id = ?", tagID).
			UpdateColumn("usage_count", gorm.Expr("(SELECT COUNT(1) FROM inspiration_post_tags WHERE tag_id = ?)", tagID)).Error; err != nil {
			return err
		}
	}
	return nil
}

func getPostTagNameMap(postIDs []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	var rows []postTagRow
	err := db.DB.Table("inspiration_post_tags AS ipt").
		Select("ipt.post_id AS post_id, it.name AS name").
		Joins("JOIN inspiration_tags it ON it.id = ipt.tag_id").
		Where("ipt.post_id IN ?", postIDs).
		Where("it.status = ?", "active").
		Order("it.usage_count DESC").
		Order("it.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.PostID] = append(result[row.PostID], row.Name)
	}
	return result, nil
}

func createPostWithRetries(tx *gorm.DB, post *db.InspirationPost) error {
	var createErr error
	for i := 0; i < 5; i++ {
		shareID, err := newShareID(12)
		if err != nil {
			return err
		}
		post.ShareID = shareID
		createErr = tx.Create(post).Error
		if createErr == nil {
			return nil
		}
		if !strings.Contains(strings.ToLower(createErr.Error()), "duplicate") {
			return createErr
		}
	}
	if createErr != nil {
		return createErr
	}
	return errors.New("failed to create post")
}

func buildInspirationPostResponse(post db.InspirationPost, author db.User, isLiked bool, tags []string) InspirationPostResponse {
	publishedAt := post.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = post.CreatedAt
	}

	sourceGenerationID := uint64(0)
	if post.SourceGenerationID != nil {
		sourceGenerationID = *post.SourceGenerationID
	}
	if tags == nil {
		tags = []string{}
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
		coverURL = deriveCoverURL(images, videoURL, "")
	}
	reviewedAtUnix := int64(0)
	if post.ReviewedAt != nil && !post.ReviewedAt.IsZero() {
		reviewedAtUnix = post.ReviewedAt.UnixMilli()
	}

	return InspirationPostResponse{
		ID:                 post.ID,
		ShareID:            post.ShareID,
		Type:               post.Type,
		SourceType:         post.SourceType,
		Title:              post.Title,
		Description:        post.Description,
		Prompt:             post.Prompt,
		Tags:               tags,
		Params:             parseJSONStringMap(post.Params),
		ReferenceImages:    parseJSONStringArray(post.ReferenceImages),
		Images:             images,
		VideoURL:           videoURL,
		CoverURL:           coverURL,
		SourceGenerationID: sourceGenerationID,
		ViewCount:          post.ViewCount,
		LikeCount:          post.LikeCount,
		RemixCount:         post.RemixCount,
		ReviewStatus:       post.ReviewStatus,
		ReviewedBySource:   post.ReviewedBySource,
		ReviewedByID:       post.ReviewedByID,
		ReviewedAt:         reviewedAtUnix,
		IsLiked:            isLiked,
		PublishedAt:        publishedAt.UnixMilli(),
		Author: InspirationAuthorResponse{
			UserID:   author.ID,
			Nickname: author.Nickname,
			Avatar:   author.Avatar,
		},
	}
}

func getGenerationForSharing(userID, generationID uint64) (*db.Generation, error) {
	var gen db.Generation
	if err := db.DB.Where("id = ? AND user_id = ?", generationID, userID).First(&gen).Error; err != nil {
		return nil, err
	}
	if gen.Status != "success" {
		return nil, errors.New("only successful generations can be shared")
	}
	images := parseJSONStringArray(gen.Images)
	if len(images) == 0 && gen.VideoURL == "" {
		return nil, errors.New("generation has no publishable media")
	}
	return &gen, nil
}

func resolveCoverURL(gen *db.Generation) string {
	images := parseJSONStringArray(gen.Images)
	if len(images) > 0 {
		return images[0]
	}
	params := parseJSONStringMap(gen.Params)
	if cover, ok := params["coverUrl"].(string); ok && cover != "" {
		return cover
	}
	if cover, ok := params["cover_url"].(string); ok && cover != "" {
		return cover
	}
	if cover, ok := params["videoCoverUrl"].(string); ok && cover != "" {
		return cover
	}
	return gen.VideoURL
}

func deriveCoverURL(images []string, videoURL, explicitCover string) string {
	explicitCover = strings.TrimSpace(explicitCover)
	if explicitCover != "" {
		return explicitCover
	}
	for _, imageURL := range images {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL != "" {
			return imageURL
		}
	}
	return strings.TrimSpace(videoURL)
}

func normalizeMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimRight(strings.ToLower(raw), "/")
}

func isVideoMediaURL(raw string) bool {
	path := normalizeMediaURL(raw)
	if path == "" {
		return false
	}
	return strings.HasSuffix(path, ".mp4") ||
		strings.HasSuffix(path, ".mov") ||
		strings.HasSuffix(path, ".webm") ||
		strings.HasSuffix(path, ".m4v") ||
		strings.HasSuffix(path, ".m3u8")
}

func isSameMediaURL(a, b string) bool {
	na := normalizeMediaURL(a)
	nb := normalizeMediaURL(b)
	return na != "" && na == nb
}

func validateVideoCoverURL(coverURL, videoURL string) error {
	coverURL = strings.TrimSpace(coverURL)
	if coverURL == "" {
		return errors.New("cover_url is required for video type")
	}
	if isSameMediaURL(coverURL, videoURL) || isVideoMediaURL(coverURL) {
		return errors.New("cover_url must be an image url")
	}
	return nil
}

func normalizeInspirationPostType(raw string) string {
	if strings.ToLower(strings.TrimSpace(raw)) == "video" {
		return "video"
	}
	return "image"
}

func buildGenerationMedia(gen *db.Generation) (postType string, mediaURLs []string) {
	images := parseJSONStringArray(gen.Images)
	videoURL := strings.TrimSpace(gen.VideoURL)

	postType = normalizeInspirationPostType(gen.Type)
	if postType == "video" {
		if videoURL != "" {
			return "video", []string{videoURL}
		}
		if len(images) > 0 {
			return "image", images
		}
		return "video", []string{}
	}

	if len(images) > 0 {
		return "image", images
	}
	if videoURL != "" {
		return "video", []string{videoURL}
	}
	return "image", []string{}
}

func getOptionalAuthedUserID(c *gin.Context) uint64 {
	if userID := c.GetUint64("userID"); userID > 0 {
		return userID
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return 0
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenString == "" {
		return 0
	}

	claims, err := auth.ValidateUserToken(tokenString)
	if err != nil {
		return 0
	}

	var user db.User
	if err := db.DB.Select("id", "status").First(&user, claims.UserID).Error; err != nil {
		return 0
	}
	if user.Status != "" && user.Status != "active" {
		return 0
	}

	return claims.UserID
}

func getLikedPostIDMap(userID uint64, postIDs []uint64) (map[uint64]struct{}, error) {
	if userID == 0 || len(postIDs) == 0 {
		return map[uint64]struct{}{}, nil
	}

	var likes []db.InspirationLike
	if err := db.DB.Select("post_id").Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&likes).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]struct{}, len(likes))
	for _, like := range likes {
		result[like.PostID] = struct{}{}
	}
	return result, nil
}

func findPublishedPostByShareID(shareID string) (*db.InspirationPost, error) {
	var post db.InspirationPost
	if err := db.DB.Where("share_id = ? AND status = ? AND review_status = ?", shareID, "published", "approved").First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
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

func listInspirationsInternal(c *gin.Context, likedOnly bool, userID uint64) {
	limit, offset := parseListPagination(c)
	postType := c.DefaultQuery("type", "all")
	tagFilter := strings.TrimSpace(c.Query("tag"))
	keyword := strings.TrimSpace(c.Query("q"))

	query := db.DB.Model(&db.InspirationPost{}).Where("inspiration_posts.status = ? AND inspiration_posts.review_status = ?", "published", "approved")
	if postType != "" && postType != "all" {
		query = query.Where("inspiration_posts.type = ?", postType)
	}
	if tagFilter != "" {
		query = query.Joins(
			"JOIN inspiration_post_tags ipt_filter ON ipt_filter.post_id = inspiration_posts.id",
		).Joins(
			"JOIN inspiration_tags it_filter ON it_filter.id = ipt_filter.tag_id",
		).Where(
			"(it_filter.slug = ? OR it_filter.name = ?) AND it_filter.status = ?",
			tagFilter,
			tagFilter,
			"active",
		)
	}
	if keyword != "" {
		likeExpr := "%" + keyword + "%"
		query = query.Where("(inspiration_posts.title LIKE ? OR inspiration_posts.prompt LIKE ?)", likeExpr, likeExpr)
	}
	if likedOnly {
		query = query.Joins(
			"JOIN inspiration_likes ON inspiration_likes.post_id = inspiration_posts.id AND inspiration_likes.user_id = ?",
			userID,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	var posts []db.InspirationPost
	if err := query.Order("inspiration_posts.published_at DESC").Order("inspiration_posts.id DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	userIDSet := map[uint64]struct{}{}
	postIDs := make([]uint64, 0, len(posts))
	for _, post := range posts {
		userIDSet[post.UserID] = struct{}{}
		postIDs = append(postIDs, post.ID)
	}

	userIDs := make([]uint64, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
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

	likedMap, err := getLikedPostIDMap(userID, postIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query likes"})
		return
	}

	tagMap, err := getPostTagNameMap(postIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query tags"})
		return
	}

	items := make([]InspirationPostResponse, 0, len(posts))
	for _, post := range posts {
		author := authors[post.UserID]
		if author.ID == 0 {
			author = db.User{ID: post.UserID, Nickname: "Creator"}
		}
		_, isLiked := likedMap[post.ID]
		items = append(items, buildInspirationPostResponse(post, author, isLiked, tagMap[post.ID]))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ShareGeneration publishes a user's generation into the inspiration feed.
func ShareGeneration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	generationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation id"})
		return
	}

	var req ShareGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	gen, err := getGenerationForSharing(userID, generationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "current generation cannot be shared"})
		return
	}

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	normalizedTags, err := sanitizeTags(req.Tags)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tags"})
		return
	}

	postPrompt := strings.TrimSpace(req.Prompt)
	if postPrompt == "" {
		postPrompt = gen.Prompt
	}

	postType, mediaURLs := buildGenerationMedia(gen)
	postCover := strings.TrimSpace(req.CoverURL)
	if postCover == "" {
		postCover = resolveCoverURL(gen)
	}
	if postType == "video" {
		videoSourceURL := ""
		if len(mediaURLs) > 0 {
			videoSourceURL = mediaURLs[0]
		}
		if err := validateVideoCoverURL(postCover, videoSourceURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	now := time.Now()
	review := buildInitialReviewSnapshot(now)
	sourceGenerationID := generationID
	savedPost := db.InspirationPost{}
	savedTags := []string{}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		post := db.InspirationPost{}
		wasNew := false
		previousReviewStatus := ""
		findErr := tx.Where("source_generation_id = ? AND user_id = ?", generationID, userID).First(&post).Error
		if findErr == nil {
			previousReviewStatus = post.ReviewStatus
			post.SourceType = "generation"
			post.Type = postType
			post.Title = req.Title
			post.Description = req.Description
			post.Prompt = postPrompt
			post.Params = gen.Params
			post.ReferenceImages = gen.ReferenceImages
			post.MediaURLs = toJSONStringArray(mediaURLs)
			post.CoverURL = postCover
			post.Status = "published"
			applyReviewSnapshot(&post, review)
			post.PublishedAt = now
			post.UpdatedAt = now
			if err := tx.Save(&post).Error; err != nil {
				return err
			}
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			wasNew = true
			post = db.InspirationPost{
				UserID:             userID,
				SourceGenerationID: &sourceGenerationID,
				SourceType:         "generation",
				Type:               postType,
				Title:              req.Title,
				Description:        req.Description,
				Prompt:             postPrompt,
				Params:             gen.Params,
				ReferenceImages:    gen.ReferenceImages,
				MediaURLs:          toJSONStringArray(mediaURLs),
				CoverURL:           postCover,
				Status:             "published",
				PublishedAt:        now,
			}
			applyReviewSnapshot(&post, review)
			if err := createPostWithRetries(tx, &post); err != nil {
				return err
			}
		} else {
			return findErr
		}

		tags, err := upsertTags(tx, normalizedTags)
		if err != nil {
			return err
		}
		if err := syncPostTags(tx, post.ID, tags); err != nil {
			return err
		}
		if wasNew || previousReviewStatus != post.ReviewStatus {
			if err := appendReviewLog(tx, post.ID, "submit", previousReviewStatus, post.ReviewStatus, "", userID); err != nil {
				return err
			}
		}

		savedPost = post
		savedTags = make([]string, 0, len(tags))
		for _, tag := range tags {
			savedTags = append(savedTags, tag.Name)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to share"})
		return
	}

	c.JSON(http.StatusOK, buildInspirationPostResponse(savedPost, *user, false, savedTags))
}

// PublishInspiration publishes a new post from uploaded media.
func PublishInspiration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	var req PublishInspirationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	sourceType := strings.ToLower(strings.TrimSpace(req.SourceType))
	if sourceType == "" {
		sourceType = "upload"
	}
	if sourceType != "upload" && sourceType != "generation" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_type"})
		return
	}

	user, ok := getActiveUser(c, userID)
	if !ok {
		return
	}

	normalizedTags, err := sanitizeTags(req.Tags)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tags"})
		return
	}

	if sourceType == "generation" {
		if req.GenerationID == nil || *req.GenerationID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "generation_id is required for generation source"})
			return
		}

		gen, err := getGenerationForSharing(userID, *req.GenerationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "current generation cannot be published"})
			return
		}

		postPrompt := strings.TrimSpace(req.Prompt)
		if postPrompt == "" {
			postPrompt = gen.Prompt
		}

		postType, mediaURLs := buildGenerationMedia(gen)
		postCover := strings.TrimSpace(req.CoverURL)
		if postCover == "" {
			postCover = resolveCoverURL(gen)
		}
		if postType == "video" {
			videoSourceURL := ""
			if len(mediaURLs) > 0 {
				videoSourceURL = mediaURLs[0]
			}
			if err := validateVideoCoverURL(postCover, videoSourceURL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		sourceGenerationID := *req.GenerationID
		now := time.Now()
		review := buildInitialReviewSnapshot(now)
		savedPost := db.InspirationPost{}
		savedTags := []string{}

		err = db.DB.Transaction(func(tx *gorm.DB) error {
			post := db.InspirationPost{}
			wasNew := false
			previousReviewStatus := ""
			findErr := tx.Where("source_generation_id = ? AND user_id = ?", sourceGenerationID, userID).First(&post).Error
			if findErr == nil {
				previousReviewStatus = post.ReviewStatus
				post.SourceType = "generation"
				post.Type = postType
				post.Title = req.Title
				post.Description = req.Description
				post.Prompt = postPrompt
				post.Params = gen.Params
				post.ReferenceImages = gen.ReferenceImages
				post.MediaURLs = toJSONStringArray(mediaURLs)
				post.CoverURL = postCover
				post.Status = "published"
				applyReviewSnapshot(&post, review)
				post.PublishedAt = now
				post.UpdatedAt = now
				if err := tx.Save(&post).Error; err != nil {
					return err
				}
			} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
				wasNew = true
				post = db.InspirationPost{
					UserID:             userID,
					SourceGenerationID: &sourceGenerationID,
					SourceType:         "generation",
					Type:               postType,
					Title:              req.Title,
					Description:        req.Description,
					Prompt:             postPrompt,
					Params:             gen.Params,
					ReferenceImages:    gen.ReferenceImages,
					MediaURLs:          toJSONStringArray(mediaURLs),
					CoverURL:           postCover,
					Status:             "published",
					PublishedAt:        now,
				}
				applyReviewSnapshot(&post, review)
				if err := createPostWithRetries(tx, &post); err != nil {
					return err
				}
			} else {
				return findErr
			}

			tags, err := upsertTags(tx, normalizedTags)
			if err != nil {
				return err
			}
			if err := syncPostTags(tx, post.ID, tags); err != nil {
				return err
			}
			if wasNew || previousReviewStatus != post.ReviewStatus {
				if err := appendReviewLog(tx, post.ID, "submit", previousReviewStatus, post.ReviewStatus, "", userID); err != nil {
					return err
				}
			}

			savedPost = post
			savedTags = make([]string, 0, len(tags))
			for _, tag := range tags {
				savedTags = append(savedTags, tag.Name)
			}
			return nil
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish"})
			return
		}

		c.JSON(http.StatusOK, buildInspirationPostResponse(savedPost, *user, false, savedTags))
		return
	}

	images := make([]string, 0, len(req.Images))
	for _, imageURL := range req.Images {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL != "" {
			images = append(images, imageURL)
		}
	}
	videoURL := strings.TrimSpace(req.VideoURL)
	if len(images) == 0 && videoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one image or video is required"})
		return
	}

	postType := strings.ToLower(strings.TrimSpace(req.Type))
	if postType == "" {
		if videoURL != "" {
			postType = "video"
		} else {
			postType = "image"
		}
	}
	if postType != "image" && postType != "video" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post type"})
		return
	}
	postType = normalizeInspirationPostType(postType)

	mediaURLs := []string{}
	if postType == "video" {
		if videoURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "video_url is required for video type"})
			return
		}
		mediaURLs = []string{videoURL}
	} else {
		if len(images) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one image is required for image type"})
			return
		}
		mediaURLs = images
	}

	postPrompt := strings.TrimSpace(req.Prompt)
	if postPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	now := time.Now()
	review := buildInitialReviewSnapshot(now)
	postCover := deriveCoverURL(images, videoURL, req.CoverURL)
	if postType == "video" {
		if err := validateVideoCoverURL(postCover, videoURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	post := db.InspirationPost{
		UserID:          userID,
		SourceType:      "upload",
		Type:            postType,
		Title:           req.Title,
		Description:     req.Description,
		Prompt:          postPrompt,
		Params:          toJSONStringMap(req.Params),
		ReferenceImages: toJSONStringArray(req.ReferenceImages),
		MediaURLs:       toJSONStringArray(mediaURLs),
		CoverURL:        postCover,
		Status:          "published",
		PublishedAt:     now,
	}
	applyReviewSnapshot(&post, review)

	savedTags := []string{}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := createPostWithRetries(tx, &post); err != nil {
			return err
		}

		tags, err := upsertTags(tx, normalizedTags)
		if err != nil {
			return err
		}
		if err := syncPostTags(tx, post.ID, tags); err != nil {
			return err
		}
		if err := appendReviewLog(tx, post.ID, "submit", "", post.ReviewStatus, "", userID); err != nil {
			return err
		}

		savedTags = make([]string, 0, len(tags))
		for _, tag := range tags {
			savedTags = append(savedTags, tag.Name)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish"})
		return
	}

	c.JSON(http.StatusOK, buildInspirationPostResponse(post, *user, false, savedTags))
}

// ListInspirationTags returns normalized tags for quick picking.
func ListInspirationTags(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	query := db.DB.Model(&db.InspirationTag{}).Where("status = ?", "active")
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var tags []db.InspirationTag
	if err := query.Order("usage_count DESC").Order("name ASC").Limit(limit).Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query tags"})
		return
	}

	items := make([]InspirationTagResponse, 0, len(tags))
	for _, tag := range tags {
		items = append(items, InspirationTagResponse{
			Name: tag.Name,
			Slug: tag.Slug,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// UnshareGeneration hides a shared generation from the public inspiration feed.
func UnshareGeneration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	generationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation id"})
		return
	}

	res := db.DB.Model(&db.InspirationPost{}).
		Where("source_generation_id = ? AND user_id = ?", generationID, userID).
		Updates(map[string]interface{}{"status": "hidden", "updated_at": time.Now()})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unshare"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "share record not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// UnshareInspirationByShareID hides a shared post by its public share id.
// This path is independent from generations and still works even if source generation is deleted.
func UnshareInspirationByShareID(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	shareID := strings.TrimSpace(c.Param("shareID"))
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	res := db.DB.Model(&db.InspirationPost{}).
		Where("share_id = ? AND user_id = ?", shareID, userID).
		Updates(map[string]interface{}{"status": "hidden", "updated_at": time.Now()})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unshare"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "share record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ListPublicInspirations returns published posts for the Explore page.
func ListPublicInspirations(c *gin.Context) {
	userID := getOptionalAuthedUserID(c)
	listInspirationsInternal(c, false, userID)
}

// ListLikedInspirations returns inspiration posts liked by current user.
func ListLikedInspirations(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}
	listInspirationsInternal(c, true, userID)
}

// ListMyInspirations returns published inspiration posts created by current user.
func ListMyInspirations(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	limit, offset := parseListPagination(c)
	postType := c.DefaultQuery("type", "all")
	tagFilter := strings.TrimSpace(c.Query("tag"))
	keyword := strings.TrimSpace(c.Query("q"))

	query := db.DB.Model(&db.InspirationPost{}).
		Where("inspiration_posts.status = ? AND inspiration_posts.user_id = ?", "published", userID)
	if postType != "" && postType != "all" {
		query = query.Where("inspiration_posts.type = ?", postType)
	}
	if tagFilter != "" {
		query = query.Joins(
			"JOIN inspiration_post_tags ipt_filter ON ipt_filter.post_id = inspiration_posts.id",
		).Joins(
			"JOIN inspiration_tags it_filter ON it_filter.id = ipt_filter.tag_id",
		).Where(
			"(it_filter.slug = ? OR it_filter.name = ?) AND it_filter.status = ?",
			tagFilter,
			tagFilter,
			"active",
		)
	}
	if keyword != "" {
		likeExpr := "%" + keyword + "%"
		query = query.Where("(inspiration_posts.title LIKE ? OR inspiration_posts.prompt LIKE ?)", likeExpr, likeExpr)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	var posts []db.InspirationPost
	if err := query.Order("inspiration_posts.published_at DESC").Order("inspiration_posts.id DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query inspirations"})
		return
	}

	postIDs := make([]uint64, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}

	likedMap, err := getLikedPostIDMap(userID, postIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query likes"})
		return
	}

	tagMap, err := getPostTagNameMap(postIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query tags"})
		return
	}

	var author db.User
	if err := db.DB.Select("id", "nickname", "avatar").First(&author, userID).Error; err != nil {
		author = db.User{ID: userID, Nickname: "Creator"}
	}

	items := make([]InspirationPostResponse, 0, len(posts))
	for _, post := range posts {
		_, isLiked := likedMap[post.ID]
		items = append(items, buildInspirationPostResponse(post, author, isLiked, tagMap[post.ID]))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetPublicInspiration returns a single published inspiration post.
func GetPublicInspiration(c *gin.Context) {
	shareID := c.Param("shareID")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	post, err := findPublishedPostByShareID(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}

	_ = db.DB.Model(&db.InspirationPost{}).
		Where("id = ?", post.ID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
	post.ViewCount++

	var author db.User
	if err := db.DB.Select("id", "nickname", "avatar").First(&author, post.UserID).Error; err != nil {
		author = db.User{ID: post.UserID, Nickname: "Creator"}
	}

	userID := getOptionalAuthedUserID(c)
	isLiked := false
	if userID > 0 {
		var count int64
		if err := db.DB.Model(&db.InspirationLike{}).Where("user_id = ? AND post_id = ?", userID, post.ID).Count(&count).Error; err == nil {
			isLiked = count > 0
		}
	}

	tagMap, err := getPostTagNameMap([]uint64{post.ID})
	if err != nil {
		tagMap = map[uint64][]string{}
	}

	c.JSON(http.StatusOK, buildInspirationPostResponse(*post, author, isLiked, tagMap[post.ID]))
}

// GetInspirationLikeStatus returns whether current user liked the post.
func GetInspirationLikeStatus(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	shareID := c.Param("shareID")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	post, err := findPublishedPostByShareID(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}

	var count int64
	if err := db.DB.Model(&db.InspirationLike{}).Where("user_id = ? AND post_id = ?", userID, post.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query like status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":      count > 0,
		"like_count": post.LikeCount,
	})
}

// LikeInspiration likes a published inspiration post.
func LikeInspiration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	shareID := c.Param("shareID")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	post, err := findPublishedPostByShareID(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}

	var likeCount int
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		like := db.InspirationLike{
			UserID:    userID,
			PostID:    post.ID,
			CreatedAt: time.Now(),
		}

		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&like)
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			if err := tx.Model(&db.InspirationPost{}).Where("id = ?", post.ID).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
				return err
			}
		}

		var refreshed db.InspirationPost
		if err := tx.Select("id", "like_count").First(&refreshed, post.ID).Error; err != nil {
			return err
		}
		likeCount = refreshed.LikeCount
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":      true,
		"like_count": likeCount,
	})
}

// UnlikeInspiration unlikes a published inspiration post.
func UnlikeInspiration(c *gin.Context) {
	userID := c.GetUint64("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login first"})
		return
	}

	shareID := c.Param("shareID")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	post, err := findPublishedPostByShareID(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}

	var likeCount int
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND post_id = ?", userID, post.ID).Delete(&db.InspirationLike{})
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			if err := tx.Model(&db.InspirationPost{}).Where("id = ?", post.ID).
				UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
				return err
			}
		}

		var refreshed db.InspirationPost
		if err := tx.Select("id", "like_count").First(&refreshed, post.ID).Error; err != nil {
			return err
		}
		likeCount = refreshed.LikeCount
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":      false,
		"like_count": likeCount,
	})
}

// MarkInspirationRemix increments remix counter for analytics.
func MarkInspirationRemix(c *gin.Context) {
	shareID := c.Param("shareID")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	res := db.DB.Model(&db.InspirationPost{}).
		Where("share_id = ? AND status = ? AND review_status = ?", shareID, "published", "approved").
		UpdateColumn("remix_count", gorm.Expr("remix_count + ?", 1))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "operation failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
