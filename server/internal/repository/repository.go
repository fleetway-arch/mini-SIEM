package repository

import (
	"context"

	"siem-dashboard/server/internal/detection"
	"siem-dashboard/server/internal/models"
)

type Repository interface {
	FindUserByEmail(context.Context, string) (models.UserDocument, error)
	FindUserByID(ctx context.Context, id, companyID int) (models.UserDocument, error)
	GetMember(context.Context, int) (models.UserDocument, error)
	CreateAccount(context.Context, models.SignupRequest) (models.User, error)
	ListTeam(context.Context) ([]models.TeamMember, error)
	CreateInvitations(context.Context, []string, string) (models.BulkInvitationResult, []models.IssuedInvitation, error)
	GetPendingInvitation(context.Context, int) (models.InvitationDocument, error)
	ResendInvitation(context.Context, int) (models.IssuedInvitation, error)
	DeleteInvitation(context.Context, int) error
	UpdateMember(context.Context, int, map[string]string) (models.TeamMember, error)
	DeleteMember(context.Context, int) error
	GetInvitation(context.Context, string) (models.InvitationDetails, error)
	AcceptInvitation(context.Context, string, string, string) (models.User, error)
	GetSummary(context.Context, string) (models.StatsSummary, error)
	ListEvents(context.Context, models.EventQuery) (models.EventPage, error)
	InsertEvents(context.Context, []models.Event) ([]models.Event, error)
	CountFailedEvents(ctx context.Context, ip, since string) (int, error)
	RecordAlert(context.Context, detection.Finding) (models.Alert, error)
	CreateIngestKey(ctx context.Context, name, keyHash, prefix string) (models.IngestKey, error)
	ListIngestKeys(context.Context) ([]models.IngestKey, error)
	RevokeIngestKey(context.Context, int) error
	UseIngestKey(ctx context.Context, keyHash string) (models.IngestKeyDocument, error)
	GetTimeline(context.Context, string, string) ([]models.TimelinePoint, error)
	GetTopIPs(context.Context, int, string) ([]models.TopIP, error)
	ListAlerts(ctx context.Context, status, address string) ([]models.Alert, error)
	UpdateAlertStatus(context.Context, int, string) (models.Alert, error)
	ListAddresses(context.Context) ([]models.Address, error)
	CreateAddress(context.Context, models.AddressRequest) (models.Address, error)
	DeleteAddress(context.Context, int) error
	CreateRefreshToken(context.Context, models.RefreshTokenDocument) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (models.RefreshTokenDocument, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeUserRefreshTokens(ctx context.Context, userID int) error
	CreatePasswordReset(ctx context.Context, user models.UserDocument, tokenHash string) error
	ResetPassword(ctx context.Context, tokenHash, newPassword string) (models.UserDocument, error)
	Close(context.Context) error
}
