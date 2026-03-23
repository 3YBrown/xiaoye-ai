package admin

type reviewInspirationRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject"`
	Note   string `json:"note" binding:"max=1000"`
}

type authorResponse struct {
	UserID   uint64 `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type inspirationResponse struct {
	ID               uint64         `json:"id"`
	ShareID          string         `json:"share_id"`
	Type             string         `json:"type"`
	Title            string         `json:"title"`
	Prompt           string         `json:"prompt"`
	Images           []string       `json:"images"`
	VideoURL         string         `json:"video_url"`
	CoverURL         string         `json:"cover_url"`
	ReviewStatus     string         `json:"review_status"`
	ReviewedBySource string         `json:"reviewed_by_source,omitempty"`
	ReviewedByID     string         `json:"reviewed_by_id,omitempty"`
	ReviewedAt       int64          `json:"reviewed_at,omitempty"`
	PublishedAt      int64          `json:"published_at"`
	Author           authorResponse `json:"author"`
}
