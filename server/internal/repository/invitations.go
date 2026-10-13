package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/tokens"
)

const InvitationTTL = 7 * 24 * time.Hour

var ErrInvitationUnavailable = errors.New("invitation is invalid, expired or already used")

// nextSequence returns an increasing integer id for a collection.
func (m *Mongo) nextSequence(ctx context.Context, name string) (int, error) {
	var counter struct {
		Seq int `bson:"seq"`
	}
	err := m.database.Collection("counters").FindOneAndUpdate(ctx,
		bson.M{"_id": name},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&counter)
	return counter.Seq, err
}

func pendingInvitationFilter(now time.Time) bson.M {
	return bson.M{"used": false, "expiresAt": bson.M{"$gt": now}}
}

func invitationMember(invitation models.InvitationDocument) models.TeamMember {
	return models.TeamMember{
		ID: invitation.ID, Email: invitation.Email, Role: invitation.Role,
		Status: "invited", InvitedAt: invitation.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (m *Mongo) inviter(ctx context.Context) (models.UserDocument, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.UserDocument{}, err
	}
	userID, ok := AuthenticatedUserID(ctx)
	if !ok {
		return models.UserDocument{}, ErrMissingCompanyContext
	}
	return m.FindUserByID(ctx, userID, filter[0].Value.(int))
}

func inviterName(user models.UserDocument) string {
	if user.FullName != "" {
		return user.FullName
	}
	return user.Email
}

// CreateInvitations creates one single-use invitation per new email and
// returns the plain tokens so the caller can email them. Only the token hash
// is stored.
func (m *Mongo) CreateInvitations(ctx context.Context, emails []string, role string) (models.BulkInvitationResult, []models.IssuedInvitation, error) {
	user, err := m.inviter(ctx)
	if err != nil {
		return models.BulkInvitationResult{}, nil, err
	}
	now := time.Now().UTC()
	result := models.BulkInvitationResult{Created: []models.TeamMember{}, Skipped: []models.SkippedInvitation{}}
	var issued []models.IssuedInvitation
	seen := map[string]bool{}
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		if !looksLikeEmail(email) {
			result.Skipped = append(result.Skipped, models.SkippedInvitation{Email: email, Reason: "invalid email"})
			continue
		}
		members, err := m.database.Collection("users").CountDocuments(ctx, bson.M{"companyId": user.CompanyID, "email": email})
		if err != nil {
			return result, issued, err
		}
		if members > 0 {
			result.Skipped = append(result.Skipped, models.SkippedInvitation{Email: email, Reason: "already exists"})
			continue
		}
		pendingFilter := pendingInvitationFilter(now)
		pendingFilter["companyId"] = user.CompanyID
		pendingFilter["email"] = email
		pending, err := m.database.Collection("invitations").CountDocuments(ctx, pendingFilter)
		if err != nil {
			return result, issued, err
		}
		if pending > 0 {
			result.Skipped = append(result.Skipped, models.SkippedInvitation{Email: email, Reason: "already invited"})
			continue
		}
		plain, hash, err := tokens.New()
		if err != nil {
			return result, issued, err
		}
		id, err := m.nextSequence(ctx, "invitations")
		if err != nil {
			return result, issued, err
		}
		invitation := models.InvitationDocument{
			ID: id, TokenHash: hash, CompanyID: user.CompanyID, CompanyName: user.CompanyName,
			Email: email, Role: role, InvitedBy: user.ID, CreatedAt: now, ExpiresAt: now.Add(InvitationTTL),
		}
		if _, err := m.database.Collection("invitations").InsertOne(ctx, invitation); err != nil {
			return result, issued, err
		}
		member := invitationMember(invitation)
		result.Created = append(result.Created, member)
		issued = append(issued, models.IssuedInvitation{
			Member: member, Email: email, Token: plain, Role: role,
			CompanyName: user.CompanyName, InviterName: inviterName(user),
		})
	}
	return result, issued, nil
}

// GetPendingInvitation loads a pending invitation of the authenticated company.
func (m *Mongo) GetPendingInvitation(ctx context.Context, id int) (models.InvitationDocument, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.InvitationDocument{}, err
	}
	query := pendingInvitationFilter(time.Now().UTC())
	query["companyId"] = filter[0].Value
	query["id"] = id
	var invitation models.InvitationDocument
	err = m.database.Collection("invitations").FindOne(ctx, query).Decode(&invitation)
	return invitation, err
}

// ResendInvitation replaces the token (the previous link stops working) and
// restarts the 7-day validity.
func (m *Mongo) ResendInvitation(ctx context.Context, id int) (models.IssuedInvitation, error) {
	user, err := m.inviter(ctx)
	if err != nil {
		return models.IssuedInvitation{}, err
	}
	plain, hash, err := tokens.New()
	if err != nil {
		return models.IssuedInvitation{}, err
	}
	now := time.Now().UTC()
	query := pendingInvitationFilter(now)
	query["companyId"] = user.CompanyID
	query["id"] = id
	var invitation models.InvitationDocument
	err = m.database.Collection("invitations").FindOneAndUpdate(ctx, query,
		bson.M{"$set": bson.M{"tokenHash": hash, "createdAt": now, "expiresAt": now.Add(InvitationTTL), "invitedBy": user.ID}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&invitation)
	if err != nil {
		return models.IssuedInvitation{}, err
	}
	return models.IssuedInvitation{
		Member: invitationMember(invitation), Email: invitation.Email, Token: plain, Role: invitation.Role,
		CompanyName: invitation.CompanyName, InviterName: inviterName(user),
	}, nil
}

func (m *Mongo) DeleteInvitation(ctx context.Context, id int) error {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return err
	}
	filter = append(filter, bson.E{Key: "id", Value: id}, bson.E{Key: "used", Value: false})
	result, err := m.database.Collection("invitations").DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// GetInvitation is public: the token itself is the credential. Expired, used
// and unknown tokens are indistinguishable.
func (m *Mongo) GetInvitation(ctx context.Context, token string) (models.InvitationDetails, error) {
	query := pendingInvitationFilter(time.Now().UTC())
	query["tokenHash"] = tokens.Hash(token)
	var invitation models.InvitationDocument
	if err := m.database.Collection("invitations").FindOne(ctx, query).Decode(&invitation); err != nil {
		return models.InvitationDetails{}, err
	}
	return models.InvitationDetails{CompanyName: invitation.CompanyName, Email: invitation.Email, Role: invitation.Role}, nil
}

// AcceptInvitation claims the invitation atomically (so a token can only be
// used once even under concurrent requests), then creates the user. If the
// user cannot be created the claim is released.
func (m *Mongo) AcceptInvitation(ctx context.Context, token, fullName, plainPassword string) (models.User, error) {
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return models.User{}, err
	}
	now := time.Now().UTC()
	query := pendingInvitationFilter(now)
	query["tokenHash"] = tokens.Hash(token)
	collection := m.database.Collection("invitations")
	var invitation models.InvitationDocument
	if err := collection.FindOneAndUpdate(ctx, query, bson.M{"$set": bson.M{"used": true, "usedAt": now}}).Decode(&invitation); err != nil {
		return models.User{}, err
	}
	user := models.UserDocument{
		User: models.User{
			ID: int(time.Now().UnixNano()), Email: invitation.Email, FullName: fullName,
			CompanyID: invitation.CompanyID, CompanyName: invitation.CompanyName, Role: invitation.Role,
		},
		PasswordHash: hash, Status: "active",
	}
	if _, err := m.database.Collection("users").InsertOne(ctx, user); err != nil {
		_, _ = collection.UpdateOne(ctx, bson.M{"id": invitation.ID, "companyId": invitation.CompanyID},
			bson.M{"$set": bson.M{"used": false}, "$unset": bson.M{"usedAt": ""}})
		if mongo.IsDuplicateKeyError(err) {
			return models.User{}, ErrInvitationUnavailable
		}
		return models.User{}, err
	}
	return user.User, nil
}

func looksLikeEmail(value string) bool {
	local, domain, found := strings.Cut(value, "@")
	return found && local != "" && strings.Contains(domain, ".") && !strings.ContainsAny(value, " \t\r\n<>,;\"")
}
