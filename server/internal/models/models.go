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

type Alert struct {
	ID          int    `json:"id" bson:"id"`
	CreatedAt   string `json:"createdAt" bson:"createdAt"`
	Rule        string `json:"rule" bson:"rule"`
	Severity    string `json:"severity" bson:"severity"`
	IP          string `json:"ip" bson:"ip"`
	Description string `json:"description" bson:"description"`
	EventCount  int    `json:"eventCount" bson:"eventCount"`
	Status      string `json:"status" bson:"status"`
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
	User
	PasswordHash string `json:"-" bson:"passwordHash"`
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

type InvitationDocument struct {
	Token       string    `bson:"token"`
	CompanyID   int       `bson:"companyId"`
	CompanyName string    `bson:"companyName"`
	Email       string    `bson:"email"`
	Role        string    `bson:"role"`
	ExpiresAt   time.Time `bson:"expiresAt"`
	Used        bool      `bson:"used"`
}
