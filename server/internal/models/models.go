package models

import "time"

type User struct {
	ID          int    `json:"id" bson:"id"`
	Email       string `json:"email" bson:"email"`
	FullName    string `json:"fullName" bson:"fullName"`
	CompanyID   int    `json:"companyId" bson:"companyId"`
	CompanyName string `json:"companyName" bson:"companyName"`
	Role        string `json:"role" bson:"role"`
}

type Event struct {
	ID        int    `json:"id" bson:"id"`
	Timestamp string `json:"timestamp" bson:"timestamp"`
	Source    string `json:"source" bson:"source"`
	Severity  string `json:"severity" bson:"severity"`
	IP        string `json:"ip" bson:"ip"`
	Message   string `json:"message" bson:"message"`
	RawLog    string `json:"rawLog" bson:"rawLog"`
}

// EventDocument is an event as stored, with the owning company.
type EventDocument struct {
	Event     `bson:",inline"`
	CompanyID int `bson:"companyId"`
}

// EventQuery holds the validated filters of GET /api/events. From and To
// are RFC 3339 UTC bounds, From inclusive and To exclusive.
type EventQuery struct {
	Page     int
	Limit    int
	Source   string
	Severity string
	IP       string
	Q        string
	From     string
	To       string
	Address  string
}

type EventPage struct {
	Items []Event `json:"items"`
	Total int64   `json:"total"`
}

type Alert struct {
	ID          int    `json:"id" bson:"id"`
	CreatedAt   string `json:"createdAt" bson:"createdAt"`
	Rule        string `json:"rule" bson:"rule"`
	Severity    string `json:"severity" bson:"severity"`
	IP          string `json:"ip" bson:"ip"`
	Description string `json:"description" bson:"description"`
	EventCount  int    `json:"eventCount" bson:"eventCount"`
	Status      string `json:"status" bson:"status"`
	LastSeenAt  string `json:"lastSeenAt,omitempty" bson:"lastSeenAt,omitempty"`
}

// IngestEvent is one event sent by a collector to POST /api/ingest/events.
type IngestEvent struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Severity  string `json:"severity"`
	IP        string `json:"ip"`
	Message   string `json:"message"`
	RawLog    string `json:"rawLog"`
}

type IngestRequest struct {
	Events []IngestEvent `json:"events"`
}

type IngestResponse struct {
	Accepted int     `json:"accepted"`
	Alerts   []Alert `json:"alerts"`
}

// IngestKey is what the API returns about an ingestion key; the plain key
// is only returned once, in CreatedIngestKey.
type IngestKey struct {
	ID         int    `json:"id" bson:"id"`
	Name       string `json:"name" bson:"name"`
	Prefix     string `json:"prefix" bson:"prefix"`
	CreatedAt  string `json:"createdAt" bson:"createdAt"`
	LastUsedAt string `json:"lastUsedAt,omitempty" bson:"lastUsedAt,omitempty"`
}

// IngestKeyDocument stores only the SHA-256 of the key.
type IngestKeyDocument struct {
	IngestKey `bson:",inline"`
	CompanyID int        `bson:"companyId"`
	KeyHash   string     `bson:"keyHash"`
	CreatedBy int        `bson:"createdBy"`
	RevokedAt *time.Time `bson:"revokedAt"`
}

type CreatedIngestKey struct {
	IngestKey
	Key string `json:"key"`
}

type IngestKeyRequest struct {
	Name string `json:"name"`
}

type StatsSummary struct {
	TotalEvents  int64 `json:"totalEvents"`
	FailedLogins int64 `json:"failedLogins"`
	UniqueIPs    int64 `json:"uniqueIps"`
	OpenAlerts   int64 `json:"openAlerts"`
}

type Address struct {
	ID        int    `json:"id" bson:"id"`
	CompanyID int    `json:"-" bson:"companyId"`
	Value     string `json:"value" bson:"value"`
	Label     string `json:"label,omitempty" bson:"label,omitempty"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
}

type TeamMember struct {
	ID          int    `json:"id" bson:"id"`
	Email       string `json:"email" bson:"email"`
	FullName    string `json:"fullName,omitempty" bson:"fullName,omitempty"`
	Role        string `json:"role" bson:"role"`
	Status      string `json:"status" bson:"status"`
	InvitedAt   string `json:"invitedAt" bson:"invitedAt"`
	LastLoginAt string `json:"lastLoginAt,omitempty" bson:"lastLoginAt,omitempty"`
}

type InvitationDetails struct {
	CompanyName string `json:"companyName"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

type BulkInvitationResult struct {
	Created []TeamMember        `json:"created"`
	Skipped []SkippedInvitation `json:"skipped"`
}

type SkippedInvitation struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

type TimelinePoint struct {
	Time   string `json:"time"`
	Total  int    `json:"total"`
	Failed int    `json:"failed"`
}

type TopIP struct {
	IP          string `json:"ip"`
	Count       int    `json:"count"`
	FailedCount int    `json:"failedCount"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Company SignupCompany `json:"company"`
	Admin   SignupAdmin   `json:"admin"`
}

type SignupCompany struct {
	Name            string `json:"name"`
	Industry        string `json:"industry"`
	Size            string `json:"size"`
	Country         string `json:"country"`
	Website         string `json:"website,omitempty"`
	SecurityContact string `json:"securityContact"`
}

type SignupAdmin struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AddressRequest struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type AlertUpdateRequest struct {
	Status string `json:"status"`
}

type UserDocument struct {
	// inline keeps id, email, companyId... at the top level of the document;
	// without it the driver nests them under a "user" key.
	User         `bson:",inline"`
	PasswordHash string `json:"-" bson:"passwordHash"`
	// Status is "active" or "revoked"; older documents without it are active.
	Status string `json:"-" bson:"status,omitempty"`
	// TokenVersion is embedded in access tokens. Incrementing it invalidates
	// every token issued before (password reset, role change, revocation).
	TokenVersion int `json:"-" bson:"tokenVersion"`
}

func (u UserDocument) IsActive() bool {
	return u.Status == "" || u.Status == "active"
}

type RefreshTokenDocument struct {
	TokenHash    string     `bson:"tokenHash"`
	UserID       int        `bson:"userId"`
	CompanyID    int        `bson:"companyId"`
	TokenVersion int        `bson:"tokenVersion"`
	CreatedAt    time.Time  `bson:"createdAt"`
	ExpiresAt    time.Time  `bson:"expiresAt"`
	RevokedAt    *time.Time `bson:"revokedAt"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
	User      User   `json:"user"`
}

type CompanyDocument struct {
	ID              int    `bson:"id"`
	Name            string `bson:"name"`
	Industry        string `bson:"industry"`
	Size            string `bson:"size"`
	Country         string `bson:"country"`
	Website         string `bson:"website,omitempty"`
	SecurityContact string `bson:"securityContact"`
	CreatedAt       string `bson:"createdAt"`
}

// InvitationDocument stores only the SHA-256 of the invitation token.
type InvitationDocument struct {
	ID          int        `bson:"id"`
	TokenHash   string     `bson:"tokenHash"`
	CompanyID   int        `bson:"companyId"`
	CompanyName string     `bson:"companyName"`
	Email       string     `bson:"email"`
	Role        string     `bson:"role"`
	InvitedBy   int        `bson:"invitedBy"`
	CreatedAt   time.Time  `bson:"createdAt"`
	ExpiresAt   time.Time  `bson:"expiresAt"`
	Used        bool       `bson:"used"`
	UsedAt      *time.Time `bson:"usedAt,omitempty"`
}

// IssuedInvitation carries the plain token from the repository to the email
// sender. It is never stored or returned to API clients.
type IssuedInvitation struct {
	Member      TeamMember
	Email       string
	Token       string
	Role        string
	CompanyName string
	InviterName string
}

// PasswordResetDocument stores only the SHA-256 of the reset token.
type PasswordResetDocument struct {
	TokenHash string     `bson:"tokenHash"`
	UserID    int        `bson:"userId"`
	CompanyID int        `bson:"companyId"`
	CreatedAt time.Time  `bson:"createdAt"`
	ExpiresAt time.Time  `bson:"expiresAt"`
	Used      bool       `bson:"used"`
	UsedAt    *time.Time `bson:"usedAt,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}
