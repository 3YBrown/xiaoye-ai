package db

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type LicenseStatus string

const (
	LicenseStatusActive   LicenseStatus = "active"
	LicenseStatusDisabled LicenseStatus = "disabled"
	LicenseStatusRedeemed LicenseStatus = "redeemed"
)

type License struct {
	ID          string        `gorm:"primaryKey;type:varchar(36);comment:License ID"`
	Balance     int           `gorm:"type:int;default:0;comment:Remaining credits"`
	Status      LicenseStatus `gorm:"type:varchar(20);default:'active';index;comment:License status"`
	ExpiresAt   *time.Time    `gorm:"type:datetime;index;comment:Expiration time"`
	UsageCount  int           `gorm:"type:int;default:0;comment:Total usage count"`
	RedeemedBy  *uint64       `gorm:"type:bigint;index;comment:User ID who redeemed this license"`
	RedeemedAt  *time.Time    `gorm:"type:datetime;comment:Redemption time"`
	OriginalKey string        `gorm:"type:text;comment:Original license key"`
	CreatedAt   time.Time     `gorm:"type:datetime;comment:Creation time"`
	UpdatedAt   time.Time     `gorm:"type:datetime;comment:Last update time"`
}

// User supports email-based auth and profile/credits data.
type User struct {
	ID              uint64     `gorm:"primaryKey;comment:User ID" json:"id"`
	Email           string     `gorm:"type:varchar(255);uniqueIndex;comment:Email address" json:"email"`
	PasswordHash    string     `gorm:"type:varchar(255);comment:Password hash" json:"-"`
	LinuxDoID       *string    `gorm:"column:linuxdo_id;type:varchar(100);uniqueIndex;comment:Linux.do OAuth ID" json:"-"`
	Nickname        string     `gorm:"type:varchar(100);comment:User nickname" json:"nickname"`
	Avatar          string     `gorm:"type:varchar(500);comment:Avatar URL" json:"avatar"`
	Credits         int        `gorm:"type:int;default:0;comment:Available credits" json:"credits"`
	TotalRedeemed   int        `gorm:"type:int;default:0;comment:Total redeemed credits" json:"total_redeemed"`
	UsageCount      int        `gorm:"type:int;default:0;comment:Total usage count" json:"usage_count"`
	Status          string     `gorm:"type:varchar(20);default:'active';index;comment:User status" json:"status"`
	EmailVerified   bool       `gorm:"type:boolean;default:false;comment:Email verified" json:"email_verified"`
	InviteCode      string     `gorm:"type:varchar(20);uniqueIndex;comment:User invite code" json:"invite_code"`
	InvitedBy       *uint64    `gorm:"type:bigint;index;comment:Inviter user ID" json:"invited_by"`
	InviteCount     int        `gorm:"type:int;default:0;comment:Number of invited users" json:"invite_count"`
	CheckinStreak   int        `gorm:"type:int;default:0;comment:Consecutive checkin days" json:"checkin_streak"`
	LastCheckinDate *time.Time `gorm:"type:date;comment:Last checkin date" json:"last_checkin_date"`
	LastLoginAt     *time.Time `gorm:"type:datetime;comment:Last login time" json:"last_login_at"`
	CreatedAt       time.Time  `gorm:"type:datetime;comment:Creation time" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"type:datetime;comment:Last update time" json:"updated_at"`
}

type EmailVerification struct {
	ID        uint64    `gorm:"primaryKey;comment:Verification ID"`
	Email     string    `gorm:"type:varchar(255);index;comment:Email address"`
	Code      string    `gorm:"type:varchar(10);comment:Verification code"`
	Type      string    `gorm:"type:varchar(20);comment:register/login/reset"`
	ExpiresAt time.Time `gorm:"type:datetime;comment:Expiration time"`
	Used      bool      `gorm:"type:boolean;default:false;comment:Whether used"`
	CreatedAt time.Time `gorm:"type:datetime;comment:Creation time"`
}

// CreditTransaction stores all credit ledger deltas.
type CreditTransaction struct {
	ID           uint64    `gorm:"primaryKey;comment:Transaction ID" json:"id"`
	UserID       uint64    `gorm:"type:bigint;index;not null;comment:User ID" json:"user_id"`
	Delta        int       `gorm:"type:int;not null;comment:Credits delta" json:"delta"`
	BalanceAfter int       `gorm:"type:int;not null;default:0;comment:Balance after tx" json:"balance_after"`
	Type         string    `gorm:"type:varchar(40);index;not null;comment:Transaction type" json:"type"`
	Source       string    `gorm:"type:varchar(40);index;comment:Business source" json:"source"`
	SourceID     string    `gorm:"type:varchar(100);index;comment:Business source ID" json:"source_id"`
	Note         string    `gorm:"type:varchar(255);comment:Note" json:"note"`
	CreatedAt    time.Time `gorm:"type:datetime;index;comment:Creation time" json:"created_at"`
}

// UserNotification stores in-app notifications for each user.
type UserNotification struct {
	ID        uint64    `gorm:"primaryKey;comment:Notification ID" json:"id"`
	UserID    uint64    `gorm:"type:bigint;index;not null;comment:Receiver user ID" json:"user_id"`
	BizKey    string    `gorm:"type:varchar(100);index;comment:Business key for idempotency" json:"biz_key"`
	Title     string    `gorm:"type:varchar(200);not null;default:'';comment:Notification title" json:"title"`
	Summary   string    `gorm:"type:varchar(500);not null;default:'';comment:Notification summary" json:"summary"`
	Content   string    `gorm:"type:text;comment:Notification content" json:"content"`
	IsRead    bool      `gorm:"type:boolean;default:false;index;comment:Is read" json:"is_read"`
	CreatedAt time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;comment:Updated at" json:"updated_at"`
}

type InvitationRecord struct {
	ID              uint64    `gorm:"primaryKey;comment:Record ID" json:"id"`
	InviterID       uint64    `gorm:"type:bigint;index;comment:Inviter user ID" json:"inviter_id"`
	InviteeID       uint64    `gorm:"type:bigint;index;comment:Invitee user ID" json:"invitee_id"`
	InviteeEmail    string    `gorm:"type:varchar(255);comment:Invitee email" json:"invitee_email"`
	CreditsRewarded int       `gorm:"type:int;default:10;comment:Credits rewarded" json:"credits_rewarded"`
	CreatedAt       time.Time `gorm:"type:datetime;comment:Invitation time" json:"created_at"`
}

func GenerateVerificationCode() string {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}

func GenerateInviteCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 8)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	return string(code)
}

type APILog struct {
	ID           uint64    `gorm:"primaryKey;comment:Log ID"`
	UserID       string    `gorm:"column:user_id;type:varchar(255);index;comment:User ID"`
	Endpoint     string    `gorm:"type:varchar(255);comment:API endpoint"`
	RequestBody  string    `gorm:"type:longtext;comment:Request body"`
	ResponseBody string    `gorm:"type:longtext;comment:Response body"`
	ResponseCode int       `gorm:"type:int;comment:HTTP response code"`
	DurationMs   int       `gorm:"type:int;comment:Duration ms"`
	CreatedAt    time.Time `gorm:"type:datetime;index;comment:Creation time"`
}

// ImageRecord is legacy and kept for compatibility.
type ImageRecord struct {
	ID           uint64    `gorm:"primaryKey;comment:Record ID" json:"id"`
	LicenseID    string    `gorm:"type:varchar(255);index;comment:License ID" json:"license_id"`
	Prompt       string    `gorm:"type:longtext;comment:Prompt" json:"prompt"`
	Mode         string    `gorm:"type:varchar(50);index;comment:Mode" json:"mode"`
	ModelID      string    `gorm:"type:varchar(100);index;comment:Model ID" json:"model_id"`
	ImageSize    string    `gorm:"type:varchar(50);comment:Image size" json:"image_size"`
	InputImages  string    `gorm:"type:text;comment:Input images(JSON)" json:"input_images"`
	OutputImages string    `gorm:"type:text;comment:Output images(JSON)" json:"output_images"`
	OutputCount  int       `gorm:"type:int;default:1;comment:Output count" json:"output_count"`
	CreditsSpent int       `gorm:"type:int;comment:Credits spent" json:"credits_spent"`
	Status       string    `gorm:"type:varchar(50);comment:Status" json:"status"`
	ErrorMessage string    `gorm:"type:text;comment:Error" json:"error_message"`
	CreatedAt    time.Time `gorm:"type:datetime;index;comment:Creation time" json:"created_at"`
}

type ImageTemplate struct {
	ID           uint64    `gorm:"primaryKey;comment:Template ID" json:"id"`
	Name         string    `gorm:"type:varchar(100);comment:Template name" json:"name"`
	Icon         string    `gorm:"type:varchar(10);comment:Template icon" json:"icon"`
	Description  string    `gorm:"type:varchar(255);comment:Template description" json:"description"`
	Prompt       string    `gorm:"type:longtext;comment:Template prompt" json:"prompt"`
	PreviewImage string    `gorm:"type:longtext;comment:Preview image URL" json:"preview_image"`
	BeforeImage  string    `gorm:"type:longtext;comment:Before image URL" json:"before_image"`
	IsActive     bool      `gorm:"type:boolean;default:true;comment:Is active" json:"is_active"`
	SortOrder    int       `gorm:"type:int;default:0;comment:Sort order" json:"sort_order"`
	CreatedAt    time.Time `gorm:"type:datetime;comment:Creation time" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:datetime;comment:Update time" json:"updated_at"`
}

// Generation is the canonical private generation history.
type Generation struct {
	ID              uint64    `gorm:"primaryKey;comment:Generation ID" json:"id"`
	UserID          uint64    `gorm:"type:bigint;index;not null;comment:User ID" json:"user_id"`
	Type            string    `gorm:"type:varchar(20);not null;default:'image';index;comment:Generation type" json:"type"`
	Prompt          string    `gorm:"type:longtext;not null;comment:Prompt" json:"prompt"`
	ReferenceImages string    `gorm:"type:text;comment:Reference image URLs(JSON)" json:"reference_images"`
	Params          string    `gorm:"type:text;comment:Params JSON" json:"params"`
	Images          string    `gorm:"type:text;comment:Output image URLs(JSON)" json:"images"`
	VideoURL        string    `gorm:"type:varchar(500);comment:Output video URL" json:"video_url"`
	Status          string    `gorm:"type:varchar(20);default:'success';index;comment:Status" json:"status"`
	CreditsCost     int       `gorm:"type:int;default:0;comment:Credits cost" json:"credits_cost"`
	ErrorMsg        string    `gorm:"type:text;comment:Error message" json:"error_msg"`
	TaskID          *string   `gorm:"type:varchar(100);index;comment:Provider task ID" json:"task_id"`
	IsFavorite      bool      `gorm:"type:boolean;default:false;index;comment:Is favorite" json:"is_favorite"`
	CreatedAt       time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"type:datetime;comment:Updated at" json:"updated_at"`
}

// PaymentOrder stores online payment orders for credit purchases.
type PaymentOrder struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	UserID          uint64     `gorm:"type:bigint;index;not null" json:"user_id"`
	OrderNo         string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"order_no"`
	Provider        string     `gorm:"type:varchar(30);index;not null" json:"provider"`
	ProviderTradeNo string     `gorm:"type:varchar(100);index" json:"provider_trade_no"`
	Amount          string     `gorm:"type:varchar(20);not null" json:"amount"`
	Diamonds        int        `gorm:"type:int;not null" json:"diamonds"`
	PlanName        string     `gorm:"type:varchar(50)" json:"plan_name"`
	Status          string     `gorm:"type:varchar(20);index;default:'pending'" json:"status"`
	NotifyData      string     `gorm:"type:text" json:"-"`
	PaidAt          *time.Time `gorm:"type:datetime" json:"paid_at"`
	CreatedAt       time.Time  `gorm:"type:datetime;index" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"type:datetime" json:"updated_at"`
}

// InspirationPost stores user-curated public posts for the inspiration feed.
type InspirationPost struct {
	ID                 uint64     `gorm:"primaryKey;comment:Inspiration post ID" json:"id"`
	ShareID            string     `gorm:"type:varchar(40);uniqueIndex;not null;comment:Public share ID" json:"share_id"`
	UserID             uint64     `gorm:"type:bigint;index;not null;comment:Author user ID" json:"user_id"`
	SourceGenerationID *uint64    `gorm:"type:bigint;uniqueIndex;comment:Source generation ID" json:"source_generation_id"`
	SourceType         string     `gorm:"type:varchar(20);index;not null;default:'generation';comment:Post source (generation/upload)" json:"source_type"`
	Type               string     `gorm:"type:varchar(20);index;not null;default:'image';comment:Generation type" json:"type"`
	Title              string     `gorm:"type:varchar(200);comment:Post title" json:"title"`
	Description        string     `gorm:"type:varchar(1000);comment:Post description" json:"description"`
	Prompt             string     `gorm:"type:longtext;comment:Public prompt" json:"prompt"`
	Params             string     `gorm:"type:text;comment:Params JSON" json:"params"`
	ReferenceImages    string     `gorm:"type:text;comment:Reference images(JSON)" json:"reference_images"`
	MediaURLs          string     `gorm:"column:media_urls;type:text;comment:Media URLs(JSON)" json:"media_urls"`
	CoverURL           string     `gorm:"type:varchar(1000);comment:Card cover URL" json:"cover_url"`
	Status             string     `gorm:"type:varchar(20);index;not null;default:'published';comment:Visibility status" json:"status"`
	ReviewStatus       string     `gorm:"type:varchar(20);index;not null;default:'approved';comment:Review status" json:"review_status"`
	ReviewedBySource   string     `gorm:"type:varchar(50);comment:Reviewer source system" json:"reviewed_by_source"`
	ReviewedByID       string     `gorm:"type:varchar(100);comment:Reviewer id in source system" json:"reviewed_by_id"`
	ReviewedAt         *time.Time `gorm:"type:datetime;comment:Review time" json:"reviewed_at"`
	ViewCount          int        `gorm:"type:int;default:0;comment:View count" json:"view_count"`
	LikeCount          int        `gorm:"type:int;default:0;comment:Like count" json:"like_count"`
	RemixCount         int        `gorm:"type:int;default:0;comment:Remix count" json:"remix_count"`
	PublishedAt        time.Time  `gorm:"type:datetime;index;comment:Published at" json:"published_at"`
	CreatedAt          time.Time  `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"type:datetime;comment:Updated at" json:"updated_at"`
}

// InspirationLike stores user-like relationships for public inspiration posts.
type InspirationLike struct {
	ID        uint64    `gorm:"primaryKey;comment:Like ID" json:"id"`
	UserID    uint64    `gorm:"type:bigint;index;not null;comment:User ID" json:"user_id"`
	PostID    uint64    `gorm:"type:bigint;index;not null;comment:Inspiration post ID" json:"post_id"`
	CreatedAt time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
}

// InspirationTag is a normalized tag dictionary entry.
type InspirationTag struct {
	ID         uint64    `gorm:"primaryKey;comment:Tag ID" json:"id"`
	Name       string    `gorm:"type:varchar(50);uniqueIndex;not null;comment:Tag display name" json:"name"`
	Slug       string    `gorm:"type:varchar(80);uniqueIndex;not null;comment:Normalized slug" json:"slug"`
	Status     string    `gorm:"type:varchar(20);index;not null;default:'active';comment:Status" json:"status"`
	UsageCount int       `gorm:"type:int;default:0;comment:Usage count" json:"usage_count"`
	CreatedAt  time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime;comment:Updated at" json:"updated_at"`
}

// InspirationPostTag is a relation row between post and tag.
type InspirationPostTag struct {
	ID        uint64    `gorm:"primaryKey;comment:Relation ID" json:"id"`
	PostID    uint64    `gorm:"type:bigint;index;not null;comment:Inspiration post ID" json:"post_id"`
	TagID     uint64    `gorm:"type:bigint;index;not null;comment:Tag ID" json:"tag_id"`
	CreatedAt time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
}

// InspirationReviewLog stores review status transitions for future moderation systems.
type InspirationReviewLog struct {
	ID             uint64    `gorm:"primaryKey;comment:Review log ID" json:"id"`
	PostID         uint64    `gorm:"type:bigint;index;not null;comment:Inspiration post ID" json:"post_id"`
	Action         string    `gorm:"type:varchar(30);index;not null;comment:Action" json:"action"`
	FromStatus     string    `gorm:"type:varchar(20);comment:From status" json:"from_status"`
	ToStatus       string    `gorm:"type:varchar(20);comment:To status" json:"to_status"`
	Note           string    `gorm:"type:varchar(1000);comment:Review note" json:"note"`
	OperatorSource string    `gorm:"type:varchar(50);comment:Operator source" json:"operator_source"`
	OperatorID     string    `gorm:"type:varchar(100);comment:Operator id in source system" json:"operator_id"`
	CreatedAt      time.Time `gorm:"type:datetime;index;comment:Created at" json:"created_at"`
}

// InitDB initializes the MySQL database connection.
func InitDB() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		log.Fatal("DB_USER environment variable is required")
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		log.Fatal("DB_NAME environment variable is required")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Printf("connecting to MySQL: %s:%s/%s", dbHost, dbPort, dbName)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if DB == nil {
		log.Fatalf("database connection returned nil")
	}

	runMigrations()
	log.Println("database initialized")
}

// runMigrations executes SQL migration files that have not been applied yet.
// It creates a schema_migrations tracking table and processes .sql files in
// sorted order from the migrations directory.
func runMigrations() {
	// Ensure tracking table exists.
	DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

	// Locate migrations directory.
	candidates := []string{
		filepath.Join(".", "migrations"),
		filepath.Join(".", "backend", "migrations"),
	}
	var migrationsDir string
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			migrationsDir = c
			break
		}
	}
	if migrationsDir == "" {
		log.Println("migrations: directory not found, skipping")
		return
	}

	// Collect .sql files sorted by name.
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Printf("migrations: failed to read directory: %v", err)
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// Load already-applied versions.
	applied := make(map[string]bool)
	var versions []string
	DB.Raw("SELECT version FROM schema_migrations").Scan(&versions)
	for _, v := range versions {
		applied[v] = true
	}

	// Execute pending migrations.
	for _, name := range files {
		if applied[name] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			log.Printf("migrations: failed to read %s: %v", name, err)
			continue
		}

		stmts := splitSQL(string(data))
		failed := false
		for _, stmt := range stmts {
			if err := DB.Exec(stmt).Error; err != nil {
				log.Printf("migrations: [%s] statement error (may be expected): %v", name, err)
				failed = true
			}
		}

		if !failed {
			DB.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name)
			log.Printf("migrations: applied %s", name)
		} else {
			log.Printf("migrations: %s had errors, not marking as applied", name)
		}
	}
}

// splitSQL splits a SQL script into individual statements by semicolons,
// ignoring empty statements and comment-only lines.
func splitSQL(content string) []string {
	parts := strings.Split(content, ";")
	var stmts []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		// Skip if all remaining lines are comments or empty.
		hasSQL := false
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "--") {
				hasSQL = true
				break
			}
		}
		if hasSQL {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

func IsLicenseActive(license *License) bool {
	if license == nil {
		return false
	}
	if license.Status != LicenseStatusActive {
		return false
	}
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

func DisableLicense(licenseID string, reason string) error {
	if err := DB.Model(&License{}).Where("id = ?", licenseID).
		Update("status", LicenseStatusDisabled).Error; err != nil {
		return fmt.Errorf("disable license failed: %w", err)
	}
	log.Printf("license disabled: %s, reason: %s", licenseID, reason)
	return nil
}

func EnableLicense(licenseID string) error {
	if err := DB.Model(&License{}).Where("id = ?", licenseID).
		Update("status", LicenseStatusActive).Error; err != nil {
		return fmt.Errorf("enable license failed: %w", err)
	}
	log.Printf("license enabled: %s", licenseID)
	return nil
}
