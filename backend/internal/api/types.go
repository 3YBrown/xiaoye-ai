package api

// SendCodeRequest sends verification code for register/login/reset.
type SendCodeRequest struct {
	Email string `json:"email"`
	Type  string `json:"type"`
}

// RegisterRequest registers a new user.
type RegisterRequest struct {
	Email      string `json:"email"`
	Code       string `json:"code"`
	Password   string `json:"password"`
	Nickname   string `json:"nickname"`
	InviteCode string `json:"invite_code"`
}

// LoginWithEmailRequest supports password or code login.
type LoginWithEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Code     string `json:"code,omitempty"`
}

// ResetPasswordRequest resets user password by email code.
type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

// RedeemKeyRequest redeems a credit key.
type RedeemKeyRequest struct {
	Key string `json:"key"`
}

// CreateGenerationRequest inserts one generation row.
type CreateGenerationRequest struct {
	Type            string                 `json:"type"`
	Prompt          string                 `json:"prompt"`
	ReferenceImages []string               `json:"reference_images,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
	Images          []string               `json:"images,omitempty"`
	VideoURL        string                 `json:"video_url,omitempty"`
	Status          string                 `json:"status,omitempty"`
	CreditsCost     int                    `json:"credits_cost,omitempty"`
	ErrorMsg        string                 `json:"error_msg,omitempty"`
	TaskID          *string                `json:"task_id,omitempty"`
}

// UpdateGenerationRequest updates a generation row.
type UpdateGenerationRequest struct {
	Images      []string `json:"images,omitempty"`
	VideoURL    string   `json:"video_url,omitempty"`
	Status      string   `json:"status,omitempty"`
	CreditsCost int      `json:"credits_cost,omitempty"`
	ErrorMsg    string   `json:"error_msg,omitempty"`
	TaskID      *string  `json:"task_id,omitempty"`
	IsFavorite  *bool    `json:"is_favorite,omitempty"`
}

// GenerationResponse is the normalized generation payload used by frontend.
type GenerationResponse struct {
	ID              uint64                 `json:"id"`
	Type            string                 `json:"type"`
	Prompt          string                 `json:"prompt"`
	ReferenceImages []string               `json:"reference_images"`
	Params          map[string]interface{} `json:"params"`
	Images          []string               `json:"images"`
	VideoURL        string                 `json:"video_url"`
	Status          string                 `json:"status"`
	CreditsCost     int                    `json:"credits_cost"`
	ErrorMsg        string                 `json:"error_msg"`
	TaskID          *string                `json:"task_id"`
	IsFavorite      bool                   `json:"is_favorite"`
	IsShared        bool                   `json:"is_shared"`
	ShareID         string                 `json:"share_id"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

// UnifiedGenerateRequest supports image/video/ecommerce generation.
type UnifiedGenerateRequest struct {
	Type   string                 `json:"type" binding:"required"`
	Prompt string                 `json:"prompt"`
	Images []string               `json:"images,omitempty"`
	Mask   string                 `json:"mask,omitempty"`
	Model  string                 `json:"model,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// PromptOptimizeRequest 提示词优化请求。
type PromptOptimizeRequest struct {
	Prompt        string                 `json:"prompt" binding:"required,max=4000"`
	CreativeMode  string                 `json:"creative_mode,omitempty"`
	Style         string                 `json:"style,omitempty"`
	TargetModel   string                 `json:"target_model,omitempty"`
	CurrentParams map[string]interface{} `json:"current_params,omitempty"`
}

// PromptOptimizeCandidate 单条优化候选结果。
type PromptOptimizeCandidate struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Prompt string `json:"prompt"`
	Reason string `json:"reason"`
}

// PromptOptimizeResponse 提示词优化响应。
type PromptOptimizeResponse struct {
	RawPrompt  string                    `json:"raw_prompt"`
	Candidates []PromptOptimizeCandidate `json:"candidates"`
	Meta       map[string]interface{}    `json:"meta,omitempty"`
}

// UploadImageRequest uploads one base64 image.
type UploadImageRequest struct {
	Image string `json:"image" binding:"required"`
}

// UploadImageResponse returns uploaded image URL.
type UploadImageResponse struct {
	URL string `json:"url"`
}

// InspirationAuthorResponse author metadata for public inspiration post.
type InspirationAuthorResponse struct {
	UserID   uint64 `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// ShareGenerationRequest publishes a generation with title and description.
type ShareGenerationRequest struct {
	Title       string   `json:"title" binding:"required,max=200"`
	Description string   `json:"description" binding:"max=1000"`
	Prompt      string   `json:"prompt,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CoverURL    string   `json:"cover_url,omitempty"`
}

// PublishInspirationRequest publishes an inspiration post from generation or upload.
type PublishInspirationRequest struct {
	SourceType      string                 `json:"source_type,omitempty"`
	GenerationID    *uint64                `json:"generation_id,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Title           string                 `json:"title" binding:"required,max=200"`
	Description     string                 `json:"description,omitempty" binding:"max=1000"`
	Prompt          string                 `json:"prompt,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Images          []string               `json:"images,omitempty"`
	VideoURL        string                 `json:"video_url,omitempty"`
	CoverURL        string                 `json:"cover_url,omitempty"`
	ReferenceImages []string               `json:"reference_images,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
}

// BindEmailRequest binds a real email to an OAuth user.
type BindEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// OAuthCallbackRequest handles the OAuth callback with code and state.
type OAuthCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// InspirationTagResponse represents one normalized tag.
type InspirationTagResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// InspirationPostResponse public inspiration post payload.
type InspirationPostResponse struct {
	ID                 uint64                    `json:"id"`
	ShareID            string                    `json:"share_id"`
	Type               string                    `json:"type"`
	SourceType         string                    `json:"source_type"`
	Title              string                    `json:"title"`
	Description        string                    `json:"description"`
	Prompt             string                    `json:"prompt"`
	Tags               []string                  `json:"tags"`
	Params             map[string]interface{}    `json:"params"`
	ReferenceImages    []string                  `json:"reference_images"`
	Images             []string                  `json:"images"`
	VideoURL           string                    `json:"video_url"`
	CoverURL           string                    `json:"cover_url"`
	SourceGenerationID uint64                    `json:"source_generation_id"`
	ViewCount          int                       `json:"view_count"`
	LikeCount          int                       `json:"like_count"`
	RemixCount         int                       `json:"remix_count"`
	ReviewStatus       string                    `json:"review_status"`
	ReviewedBySource   string                    `json:"reviewed_by_source,omitempty"`
	ReviewedByID       string                    `json:"reviewed_by_id,omitempty"`
	ReviewedAt         int64                     `json:"reviewed_at,omitempty"`
	IsLiked            bool                      `json:"is_liked"`
	PublishedAt        int64                     `json:"published_at"`
	Author             InspirationAuthorResponse `json:"author"`
}
