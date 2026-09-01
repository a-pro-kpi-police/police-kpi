package main

import (
	"database/sql"
	"time"
)

const (
	roleAdmin   = "admin"
	roleHead    = "head"
	roleOfficer = "officer"

	statusPending   = "pending"
	statusConfirmed = "confirmed"
	statusRejected  = "rejected"
)

type District struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Coeff float64 `json:"coeff"`
}

type User struct {
	ID                   int
	Username             string
	PasswordHash         string
	FullName             string
	Rank                 string
	Role                 string
	DistrictID           sql.NullInt64
	SourceEmployeeNumber sql.NullString
	MonthlyNorm          float64
	District             *District
}

type UserOut struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"full_name"`
	Rank        string    `json:"rank"`
	Role        string    `json:"role"`
	MonthlyNorm float64   `json:"monthly_norm"`
	District    *District `json:"district"`
}

type OfficerOut struct {
	ID              int       `json:"id"`
	Username        string    `json:"username"`
	FullName        string    `json:"full_name"`
	Rank            string    `json:"rank"`
	Role            string    `json:"role"`
	MonthlyNorm     float64   `json:"monthly_norm"`
	District        *District `json:"district"`
	EfficiencyRank  int       `json:"efficiency_rank"`
	EfficiencyTotal int       `json:"efficiency_total"`
	EfficiencyScore float64   `json:"efficiency_score"`
}

type AdminUserOut struct {
	ID                   int       `json:"id"`
	Username             string    `json:"username"`
	FullName             string    `json:"full_name"`
	Rank                 string    `json:"rank"`
	Role                 string    `json:"role"`
	MonthlyNorm          float64   `json:"monthly_norm"`
	District             *District `json:"district"`
	SourceEmployeeNumber string    `json:"source_employee_number"`
	EntriesCount         int       `json:"entries_count"`
}

type AdminUserSavePayload struct {
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	FullName             string   `json:"full_name"`
	Rank                 string   `json:"rank"`
	Role                 string   `json:"role"`
	DistrictID           *int     `json:"district_id"`
	SourceEmployeeNumber string   `json:"source_employee_number"`
	MonthlyNorm          *float64 `json:"monthly_norm"`
}

type AdminCatalogSavePayload struct {
	Code          string   `json:"code"`
	Indicator     string   `json:"indicator"`
	Block         string   `json:"block"`
	Subblock      string   `json:"subblock"`
	NPA           string   `json:"npa"`
	PointsPerUnit *float64 `json:"points_per_unit"`
	Unit          string   `json:"unit"`
	SourceURL     string   `json:"source_url"`
	MonthlyLimit  *float64 `json:"monthly_limit"`
	IsMass        bool     `json:"is_mass"`
	IsCritical    bool     `json:"is_critical"`
}

type RatingOfficerOut struct {
	OfficerID       int     `json:"officer_id"`
	Name            string  `json:"name"`
	Rank            string  `json:"rank"`
	District        string  `json:"district"`
	DistrictID      int     `json:"district_id"`
	Final           float64 `json:"final"`
	Confirmed       int     `json:"confirmed"`
	Pending         int     `json:"pending"`
	CategoryScore   float64 `json:"category_score"`
	EfficiencyRank  int     `json:"efficiency_rank"`
	EfficiencyTotal int     `json:"efficiency_total"`
	EfficiencyScore float64 `json:"efficiency_score"`
}

type CatalogItem struct {
	ID            int     `json:"id"`
	Code          string  `json:"code"`
	Indicator     string  `json:"indicator"`
	Block         string  `json:"block"`
	Subblock      string  `json:"subblock"`
	NPA           string  `json:"npa"`
	PointsPerUnit float64 `json:"points_per_unit"`
	Unit          string  `json:"unit"`
	SourceURL     string  `json:"source_url"`
	MonthlyLimit  float64 `json:"monthly_limit"`
	IsMass        bool    `json:"is_mass"`
	IsCritical    bool    `json:"is_critical"`
	IsActive      bool    `json:"-"`
}

type EntryOut struct {
	ID                    int     `json:"id"`
	Date                  string  `json:"date"`
	Month                 string  `json:"month"`
	OfficerID             int     `json:"officer_id"`
	OfficerName           string  `json:"officer_name"`
	Rank                  string  `json:"rank"`
	DistrictName          string  `json:"district_name"`
	Code                  string  `json:"code"`
	Indicator             string  `json:"indicator"`
	Block                 string  `json:"block"`
	Subblock              string  `json:"subblock"`
	NPA                   string  `json:"npa"`
	SourceURL             string  `json:"source_url"`
	Quantity              float64 `json:"quantity"`
	CountedQuantity       float64 `json:"counted_quantity"`
	MonthlyLimit          float64 `json:"monthly_limit"`
	IsMass                bool    `json:"is_mass"`
	IsCritical            bool    `json:"is_critical"`
	PointsPerUnit         float64 `json:"points_per_unit"`
	Coeff                 float64 `json:"coeff"`
	RawScore              float64 `json:"raw_score"`
	CappedScore           float64 `json:"capped_score"`
	AntiCoeff             float64 `json:"anti_coeff"`
	FinalScore            float64 `json:"final_score"`
	Status                string  `json:"status"`
	Comments              string  `json:"comments"`
	HeadComment           string  `json:"head_comment"`
	AttachmentName        string  `json:"attachment_name"`
	AttachmentContentType string  `json:"attachment_content_type"`
	AttachmentSize        int64   `json:"attachment_size"`
}

type TokenOut struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type EntryCreate struct {
	CatalogID int      `json:"catalog_id"`
	OfficerID *int     `json:"officer_id"`
	Quantity  *float64 `json:"quantity"`
	Date      *string  `json:"date"`
	Comments  string   `json:"comments"`
}

type EntryStatusUpdate struct {
	Status      string `json:"status"`
	HeadComment string `json:"head_comment"`
}

type TargetOut struct {
	CatalogID      int     `json:"catalog_id"`
	Code           string  `json:"code"`
	Indicator      string  `json:"indicator"`
	Block          string  `json:"block"`
	Unit           string  `json:"unit"`
	PointsPerUnit  float64 `json:"points_per_unit"`
	MonthlyLimit   float64 `json:"monthly_limit"`
	IsMass         bool    `json:"is_mass"`
	IsCritical     bool    `json:"is_critical"`
	TargetQuantity float64 `json:"target_quantity"`
	TargetScore    float64 `json:"target_score"`
}

type TargetSaveItem struct {
	CatalogID      int     `json:"catalog_id"`
	TargetQuantity float64 `json:"target_quantity"`
}

type TargetSavePayload struct {
	OfficerID int              `json:"officer_id"`
	Month     string           `json:"month"`
	Items     []TargetSaveItem `json:"items"`
}

type GroupTargetOut struct {
	TargetOut
	Mixed bool `json:"mixed"`
}

type GroupTargetSavePayload struct {
	DistrictID int              `json:"district_id"`
	Month      string           `json:"month"`
	Items      []TargetSaveItem `json:"items"`
}

type TargetProgressItemOut struct {
	CatalogID          int     `json:"catalog_id"`
	Code               string  `json:"code"`
	Indicator          string  `json:"indicator"`
	Block              string  `json:"block"`
	Unit               string  `json:"unit"`
	PointsPerUnit      float64 `json:"points_per_unit"`
	MonthlyLimit       float64 `json:"monthly_limit"`
	IsMass             bool    `json:"is_mass"`
	IsCritical         bool    `json:"is_critical"`
	TargetQuantity     float64 `json:"target_quantity"`
	ConfirmedQuantity  float64 `json:"confirmed_quantity"`
	CountedQuantity    float64 `json:"counted_quantity"`
	RemainingQuantity  float64 `json:"remaining_quantity"`
	TargetScore        float64 `json:"target_score"`
	ConfirmedScore     float64 `json:"confirmed_score"`
	AchievementPercent float64 `json:"achievement_percent"`
}

type TargetProgressOut struct {
	Month              string                  `json:"month"`
	OfficerID          int                     `json:"officer_id"`
	ActualScore        float64                 `json:"actual_score"`
	TargetScore        float64                 `json:"target_score"`
	AchievementPercent float64                 `json:"achievement_percent"`
	Items              []TargetProgressItemOut `json:"items"`
}

type TargetOverviewOfficerOut struct {
	OfficerID          int     `json:"officer_id"`
	Name               string  `json:"name"`
	ActualScore        float64 `json:"actual_score"`
	TargetScore        float64 `json:"target_score"`
	AchievementPercent float64 `json:"achievement_percent"`
	Gap                float64 `json:"gap"`
	Status             string  `json:"status"`
	PenaltyScore       float64 `json:"penalty_score"`
	PenaltyCount       int     `json:"penalty_count"`
}

type TargetBlockProgressOut struct {
	Block              string  `json:"block"`
	ActualScore        float64 `json:"actual_score"`
	TargetScore        float64 `json:"target_score"`
	AchievementPercent float64 `json:"achievement_percent"`
}

type TargetOverviewOut struct {
	Month              string                     `json:"month"`
	ActualScore        float64                    `json:"actual_score"`
	TargetScore        float64                    `json:"target_score"`
	AchievementPercent float64                    `json:"achievement_percent"`
	OnTargetCount      int                        `json:"on_target_count"`
	OfficersCount      int                        `json:"officers_count"`
	PenaltyScore       float64                    `json:"penalty_score"`
	PenaltyCount       int                        `json:"penalty_count"`
	Officers           []TargetOverviewOfficerOut `json:"officers"`
	Blocks             []TargetBlockProgressOut   `json:"blocks"`
	Items              []TargetProgressItemOut    `json:"items"`
}

type NotificationOut struct {
	ID        int       `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Link      string    `json:"link"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationListOut struct {
	UnreadCount int               `json:"unread_count"`
	Items       []NotificationOut `json:"items"`
}

type MessengerContactOut struct {
	ID                 int        `json:"id"`
	FullName           string     `json:"full_name"`
	Rank               string     `json:"rank"`
	Role               string     `json:"role"`
	DistrictName       string     `json:"district_name"`
	UnreadCount        int        `json:"unread_count"`
	LastMessage        string     `json:"last_message"`
	LastMessageAt      *time.Time `json:"last_message_at"`
	LastMessageDeleted bool       `json:"last_message_deleted"`
	EfficiencyRank     *int       `json:"efficiency_rank"`
	EfficiencyTotal    *int       `json:"efficiency_total"`
	EfficiencyScore    *float64   `json:"efficiency_score"`
}

type MessengerUnreadOut struct {
	UnreadCount int `json:"unread_count"`
}

type MessageCreate struct {
	RecipientID int    `json:"recipient_id"`
	Body        string `json:"body"`
	EntryID     *int   `json:"entry_id"`
}

type MessengerEntryOut struct {
	ID              int     `json:"id"`
	Date            string  `json:"date"`
	Code            string  `json:"code"`
	Indicator       string  `json:"indicator"`
	Block           string  `json:"block"`
	NPA             string  `json:"npa"`
	SourceURL       string  `json:"source_url"`
	Quantity        float64 `json:"quantity"`
	CountedQuantity float64 `json:"counted_quantity"`
	MonthlyLimit    float64 `json:"monthly_limit"`
	PointsPerUnit   float64 `json:"points_per_unit"`
	Coeff           float64 `json:"coeff"`
	RawScore        float64 `json:"raw_score"`
	CappedScore     float64 `json:"capped_score"`
	AntiCoeff       float64 `json:"anti_coeff"`
	FinalScore      float64 `json:"final_score"`
	Status          string  `json:"status"`
	Comments        string  `json:"comments"`
	HeadComment     string  `json:"head_comment"`
}

type MessageOut struct {
	ID          int                `json:"id"`
	SenderID    int                `json:"sender_id"`
	RecipientID int                `json:"recipient_id"`
	SenderName  string             `json:"sender_name"`
	Body        string             `json:"body"`
	CreatedAt   time.Time          `json:"created_at"`
	ReadAt      *time.Time         `json:"read_at"`
	SentByMe    bool               `json:"sent_by_me"`
	IsDeleted   bool               `json:"is_deleted"`
	Entry       *MessengerEntryOut `json:"entry"`
}

type SupportConversationUserOut struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"full_name"`
	Rank        string    `json:"rank"`
	Role        string    `json:"role"`
	District    *District `json:"district"`
	MonthlyNorm float64   `json:"monthly_norm"`
}

type SupportConversationOut struct {
	ID            int                        `json:"id"`
	User          SupportConversationUserOut `json:"user"`
	LastMessage   string                     `json:"last_message"`
	LastMessageAt *time.Time                 `json:"last_message_at"`
	MessageCount  int                        `json:"message_count"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type SupportThreadOut struct {
	Conversation *SupportConversationOut `json:"conversation"`
	Messages     []SupportMessageOut     `json:"messages"`
}

type SupportMessageCreate struct {
	Body string `json:"body"`
}

type SupportMessageOut struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	SenderID       int       `json:"sender_id"`
	SenderName     string    `json:"sender_name"`
	SenderRole     string    `json:"sender_role"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
	SentByMe       bool      `json:"sent_by_me"`
	SentByAdmin    bool      `json:"sent_by_admin"`
}
