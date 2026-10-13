package repository

import (
	"context"

	"siem-dashboard/server/internal/models"
)

type Repository interface {
	FindUserByEmail(context.Context, string) (models.UserDocument, error)
	CreateAccount(context.Context, models.SignupRequest) (models.User, error)
	ListTeam(context.Context) ([]models.TeamMember, error)
	CreateInvitations(context.Context, []string, string) (models.BulkInvitationResult, error)
	UpdateMember(context.Context, int, map[string]string) (models.TeamMember, error)
	DeleteMember(context.Context, int) error
	GetInvitation(context.Context, string) (models.InvitationDetails, error)
	AcceptInvitation(context.Context, string, string, string) (models.User, error)
	GetSummary(context.Context, string) (models.StatsSummary, error)
	ListEvents(context.Context) ([]models.Event, error)
	GetTimeline(context.Context, string, string) ([]models.TimelinePoint, error)
	GetTopIPs(context.Context, int, string) ([]models.TopIP, error)
	ListAlerts(context.Context, string) ([]models.Alert, error)
	UpdateAlertStatus(context.Context, int, string) (models.Alert, error)
	ListAddresses(context.Context) ([]models.Address, error)
	CreateAddress(context.Context, models.AddressRequest) (models.Address, error)
	DeleteAddress(context.Context, int) error
	Close(context.Context) error
}
