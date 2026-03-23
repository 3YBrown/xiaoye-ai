package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"google-ai-proxy/internal/db"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxTagCount   = 5
	maxTagRunes   = 24
	defaultSource = "upload"
)

type importItem struct {
	Title       string
	Description string
	Prompt      string
	Type        string
	MediaURL    string
	CoverURL    string
	Tags        []string
}

type normalizedTag struct {
	Name string
	Slug string
}

type postTagLink struct {
	TagID uint64 `gorm:"column:tag_id"`
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: failed to load .env from backend/: %v", err)
	}
	db.InitDB()

	inputPath, err := resolveInputPath()
	if err != nil {
		log.Fatalf("resolve import file failed: %v", err)
	}
	log.Printf("using import file: %s", inputPath)

	rawItems, err := readJSONArray(inputPath)
	if err != nil {
		log.Fatalf("read import file failed: %v", err)
	}
	if len(rawItems) == 0 {
		log.Fatalf("no valid rows found in %s", inputPath)
	}

	userID, err := resolveUserID()
	if err != nil {
		log.Fatalf("resolve user id failed: %v", err)
	}
	now := time.Now()

	var createdPosts, updatedPosts, failedRows, skippedRows int
	for i, raw := range rawItems {
		item, ok := parseRawItem(raw)
		if !ok {
			skippedRows++
			continue
		}

		shareID := buildShareID(item.MediaURL)
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			postID, created, err := upsertPost(tx, userID, shareID, item, now)
			if err != nil {
				return err
			}

			tags, err := upsertTags(tx, item.Tags, now)
			if err != nil {
				return err
			}
			if err := replacePostTags(tx, postID, tags, now); err != nil {
				return err
			}

			if created {
				createdPosts++
			} else {
				updatedPosts++
			}
			return nil
		})
		if err != nil {
			failedRows++
			log.Printf("row %d failed: %v", i+1, err)
		}
	}

	log.Printf(
		"import done: total=%d created_posts=%d updated_posts=%d skipped=%d failed=%d",
		len(rawItems), createdPosts, updatedPosts, skippedRows, failedRows,
	)
}

func resolveInputPath() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("INSP_IMPORT_FILE")); raw != "" {
		return filepath.Clean(raw), nil
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to locate current file")
	}
	srcDir := filepath.Dir(filename)
	candidates := []string{
		filepath.Join(srcDir, "full.json"),
		filepath.Join(srcDir, "prompts_full.json"),
		"full.json",
		"prompts_full.json",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}
	return "", errors.New("full.json not found (set INSP_IMPORT_FILE)")
}

func readJSONArray(path string) ([]map[string]interface{}, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return list, nil
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for _, key := range []string{"items", "data", "list", "rows"} {
		v, ok := obj[key]
		if !ok {
			continue
		}
		if arr, ok := asObjectArray(v); ok {
			return arr, nil
		}
	}

	// Fallback: scan first array-like field.
	for _, v := range obj {
		if arr, ok := asObjectArray(v); ok {
			return arr, nil
		}
	}
	return nil, errors.New("no array payload found in json file")
}

func asObjectArray(v interface{}) ([]map[string]interface{}, bool) {
	rawArr, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]map[string]interface{}, 0, len(rawArr))
	for _, item := range rawArr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, obj)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func parseRawItem(raw map[string]interface{}) (importItem, bool) {
	texts := make([]string, 0, len(raw))
	urls := make([]string, 0, len(raw))
	tagCandidates := make([][]string, 0, 2)

	for _, v := range raw {
		switch tv := v.(type) {
		case string:
			s := strings.TrimSpace(tv)
			if s == "" {
				continue
			}
			if looksLikeURL(s) {
				urls = append(urls, s)
			} else {
				texts = append(texts, s)
			}
		case []interface{}:
			tags := parseTagArray(tv)
			if len(tags) > 0 {
				tagCandidates = append(tagCandidates, tags)
			}
		}
	}

	videoURL := firstURLByType(urls, true)
	imageURL := firstURLByType(urls, false)

	item := importItem{}
	if videoURL != "" {
		item.Type = "video"
		item.MediaURL = videoURL
		if imageURL != "" {
			item.CoverURL = imageURL
		} else {
			item.CoverURL = videoURL + "#t=0.1"
		}
	} else if imageURL != "" {
		item.Type = "image"
		item.MediaURL = imageURL
		item.CoverURL = imageURL
	} else {
		return importItem{}, false
	}

	item.Title = pickTitle(texts)
	item.Prompt = pickChinesePrompt(texts, item.Title)
	if item.Prompt == "" {
		return importItem{}, false
	}
	if item.Title == "" {
		item.Title = truncateRunes(item.Prompt, 60)
	}
	item.Description = ""

	if len(tagCandidates) > 0 {
		sort.Slice(tagCandidates, func(i, j int) bool { return len(tagCandidates[i]) > len(tagCandidates[j]) })
		item.Tags = tagCandidates[0]
	}
	return item, true
}

func parseTagArray(input []interface{}) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	for _, v := range input {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		s = cleanTag(s)
		if s == "" || looksLikeURL(s) {
			return nil
		}
		if runeLen(s) > maxTagRunes {
			return nil
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return uniqueStrings(out, maxTagCount)
}

func pickTitle(texts []string) string {
	type candidate struct {
		Text string
		Len  int
		CJK  int
	}

	cands := make([]candidate, 0, len(texts))
	for _, s := range texts {
		n := runeLen(s)
		if n < 2 || n > 80 {
			continue
		}
		cands = append(cands, candidate{
			Text: s,
			Len:  n,
			CJK:  hanCount(s),
		})
	}
	if len(cands) == 0 {
		return ""
	}

	sort.Slice(cands, func(i, j int) bool {
		// Prefer Chinese short text as title, then shorter length.
		if (cands[i].CJK > 0) != (cands[j].CJK > 0) {
			return cands[i].CJK > 0
		}
		if cands[i].Len != cands[j].Len {
			return cands[i].Len < cands[j].Len
		}
		return cands[i].Text < cands[j].Text
	})
	return strings.TrimSpace(cands[0].Text)
}

func pickChinesePrompt(texts []string, title string) string {
	best := ""
	bestScore := -1
	for _, s := range texts {
		ss := strings.TrimSpace(s)
		if ss == "" || ss == title {
			continue
		}
		cjk := hanCount(ss)
		if cjk <= 0 {
			continue
		}
		score := cjk*100 + runeLen(ss)
		if score > bestScore {
			best = ss
			bestScore = score
		}
	}
	if best != "" {
		return best
	}

	// Fallback to the longest text if no Chinese prompt is detected.
	for _, s := range texts {
		ss := strings.TrimSpace(s)
		if ss == "" || ss == title {
			continue
		}
		if runeLen(ss) > runeLen(best) {
			best = ss
		}
	}
	return best
}

func firstURLByType(urls []string, wantVideo bool) string {
	for _, u := range urls {
		if wantVideo && isVideoURL(u) {
			return u
		}
		if !wantVideo && isImageURL(u) {
			return u
		}
	}
	// Fallback: if no explicit extension exists, pick any URL.
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func upsertPost(tx *gorm.DB, userID uint64, shareID string, item importItem, now time.Time) (uint64, bool, error) {
	mediaJSON := mustJSON([]string{item.MediaURL})
	paramsJSON := "{}"
	reviewedAt := now

	var post db.InspirationPost
	err := tx.Where("share_id = ?", shareID).Take(&post).Error
	if err == nil {
		updates := map[string]interface{}{
			"user_id":              userID,
			"source_generation_id": nil,
			"source_type":          defaultSource,
			"type":                 item.Type,
			"title":                item.Title,
			"description":          item.Description,
			"prompt":               item.Prompt,
			"params":               paramsJSON,
			"reference_images":     "[]",
			"media_urls":           mediaJSON,
			"cover_url":            item.CoverURL,
			"status":               "published",
			"review_status":        "approved",
			"reviewed_by_source":   "system",
			"reviewed_by_id":       "import",
			"reviewed_at":          reviewedAt,
			"published_at":         now,
			"updated_at":           now,
		}
		if err := tx.Model(&db.InspirationPost{}).Where("id = ?", post.ID).Updates(updates).Error; err != nil {
			return 0, false, err
		}
		return post.ID, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	post = db.InspirationPost{
		ShareID:            shareID,
		UserID:             userID,
		SourceGenerationID: nil,
		SourceType:         defaultSource,
		Type:               item.Type,
		Title:              item.Title,
		Description:        item.Description,
		Prompt:             item.Prompt,
		Params:             paramsJSON,
		ReferenceImages:    "[]",
		MediaURLs:          mediaJSON,
		CoverURL:           item.CoverURL,
		Status:             "published",
		ReviewStatus:       "approved",
		ReviewedBySource:   "system",
		ReviewedByID:       "import",
		ReviewedAt:         &reviewedAt,
		PublishedAt:        now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := tx.Create(&post).Error; err != nil {
		return 0, false, err
	}
	return post.ID, true, nil
}

func upsertTags(tx *gorm.DB, rawTags []string, now time.Time) ([]db.InspirationTag, error) {
	normalized := normalizeTags(rawTags)
	if len(normalized) == 0 {
		return []db.InspirationTag{}, nil
	}

	result := make([]db.InspirationTag, 0, len(normalized))
	for _, t := range normalized {
		row := db.InspirationTag{
			Name:      t.Name,
			Slug:      t.Slug,
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slug"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"name":       t.Name,
				"status":     "active",
				"updated_at": now,
			}),
		}).Create(&row).Error; err != nil {
			return nil, err
		}

		var current db.InspirationTag
		if err := tx.Where("slug = ?", t.Slug).First(&current).Error; err != nil {
			return nil, err
		}
		result = append(result, current)
	}
	return result, nil
}

func replacePostTags(tx *gorm.DB, postID uint64, tags []db.InspirationTag, now time.Time) error {
	var oldLinks []postTagLink
	if err := tx.Model(&db.InspirationPostTag{}).Select("tag_id").Where("post_id = ?", postID).Find(&oldLinks).Error; err != nil {
		return err
	}

	impacted := map[uint64]struct{}{}
	for _, l := range oldLinks {
		impacted[l.TagID] = struct{}{}
	}

	if err := tx.Where("post_id = ?", postID).Delete(&db.InspirationPostTag{}).Error; err != nil {
		return err
	}

	for _, t := range tags {
		impacted[t.ID] = struct{}{}
		link := db.InspirationPostTag{
			PostID:    postID,
			TagID:     t.ID,
			CreatedAt: now,
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

func normalizeTags(raw []string) []normalizedTag {
	if len(raw) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]normalizedTag, 0, len(raw))
	for _, tag := range raw {
		name := cleanTag(tag)
		if name == "" || runeLen(name) > maxTagRunes {
			continue
		}
		slug := normalizeTagSlug(name)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, normalizedTag{Name: name, Slug: slug})
		if len(out) >= maxTagCount {
			break
		}
	}
	return out
}

func cleanTag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.Join(strings.Fields(s), " ")
	return s
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
		if strings.ContainsRune(" -_#", r) {
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func resolveUserID() (uint64, error) {
	if raw := strings.TrimSpace(os.Getenv("INSP_IMPORT_USER_ID")); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid INSP_IMPORT_USER_ID: %w", err)
		}
		var cnt int64
		if err := db.DB.Model(&db.User{}).Where("id = ?", v).Count(&cnt).Error; err != nil {
			return 0, err
		}
		if cnt == 0 {
			return 0, fmt.Errorf("user id %d not found", v)
		}
		return v, nil
	}

	var u db.User
	if err := db.DB.Order("id ASC").First(&u).Error; err != nil {
		return 0, err
	}
	return u.ID, nil
}

func buildShareID(seed string) string {
	h := sha1.Sum([]byte(seed))
	s := hex.EncodeToString(h[:])
	return "fj" + s[:14]
}

func looksLikeURL(s string) bool {
	l := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

func isVideoURL(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, ".mp4") ||
		strings.Contains(l, ".mov") ||
		strings.Contains(l, ".webm") ||
		strings.Contains(l, ".m3u8") ||
		strings.Contains(l, ".mkv") ||
		strings.Contains(l, ".avi")
}

func isImageURL(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, ".jpg") ||
		strings.Contains(l, ".jpeg") ||
		strings.Contains(l, ".png") ||
		strings.Contains(l, ".webp") ||
		strings.Contains(l, ".gif") ||
		strings.Contains(l, ".bmp") ||
		strings.Contains(l, ".avif")
}

func hanCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			n++
		}
	}
	return n
}

func runeLen(s string) int {
	return len([]rune(s))
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func uniqueStrings(in []string, limit int) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
