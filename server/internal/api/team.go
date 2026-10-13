package api

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"siem-dashboard/server/internal/mail"
	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/repository"
)

const maxInvitationsPerRequest = 100

// canManage mirrors the frontend useCan() rules: the owner manages everyone
// except the owner, and an admin only manages viewers.
func canManage(actorRole, targetRole string) bool {
	switch actorRole {
	case roleOwner:
		return targetRole != roleOwner
	case roleAdmin:
		return targetRole == roleViewer
	default:
		return false
	}
}

func isAssignableRole(role string) bool {
	return role == roleAdmin || role == roleViewer
}

func (s *Server) listTeam(w http.ResponseWriter, r *http.Request) {
	team, err := s.repository.ListTeam(r.Context())
	if err != nil {
		internalError(w, "could not list team", err)
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (s *Server) createInvitations(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Emails []string `json:"emails"`
		Role   string   `json:"role"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !isAssignableRole(input.Role) {
		writeError(w, 400, "role must be admin or viewer")
		return
	}
	if len(input.Emails) == 0 || len(input.Emails) > maxInvitationsPerRequest {
		writeError(w, 400, "between 1 and 100 emails are required")
		return
	}
	result, issued, err := s.repository.CreateInvitations(r.Context(), input.Emails, input.Role)
	if err != nil {
		internalError(w, "could not create invitations", err)
		return
	}
	for _, invitation := range issued {
		s.sendInvitationEmail(invitation)
	}
	writeJSON(w, 200, result)
}

func (s *Server) resendInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "invalid member id")
		return
	}
	isInvitation, ok := s.authorizeMemberChange(w, r, id)
	if !ok {
		return
	}
	if !isInvitation {
		writeError(w, 400, "only pending invitations can be resent")
		return
	}
	invitation, err := s.repository.ResendInvitation(r.Context(), id)
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "member not found")
			return
		}
		internalError(w, "could not resend invitation", err)
		return
	}
	s.sendInvitationEmail(invitation)
	writeJSON(w, 200, invitation.Member)
}

// sendInvitationEmail queues the email; delivery problems are only logged.
func (s *Server) sendInvitationEmail(invitation models.IssuedInvitation) {
	message, err := mail.InvitationEmail(invitation.Email, mail.InvitationData{
		CompanyName: invitation.CompanyName,
		InviterName: invitation.InviterName,
		Role:        invitation.Role,
		Link:        s.options.AppBaseURL + "/invite/" + url.PathEscape(invitation.Token),
	})
	if err != nil {
		log.Printf("render invitation email: %v", err)
		return
	}
	s.mail.Enqueue(message)
}

// authorizeMemberChange loads the target (a member or a pending invitation)
// and checks that the caller may modify it. Otherwise it writes the error
// response and returns ok=false.
func (s *Server) authorizeMemberChange(w http.ResponseWriter, r *http.Request, id int) (isInvitation bool, ok bool) {
	actorID, _ := repository.AuthenticatedUserID(r.Context())
	actorRole, _ := repository.AuthenticatedRole(r.Context())
	if id == actorID {
		writeError(w, 403, "you cannot modify your own account")
		return false, false
	}
	targetRole := ""
	member, err := s.repository.GetMember(r.Context(), id)
	switch {
	case err == nil:
		targetRole = member.Role
	case repository.IsNotFound(err):
		invitation, invErr := s.repository.GetPendingInvitation(r.Context(), id)
		if invErr != nil {
			if repository.IsNotFound(invErr) {
				writeError(w, 404, "member not found")
			} else {
				internalError(w, "could not load member", invErr)
			}
			return false, false
		}
		targetRole, isInvitation = invitation.Role, true
	default:
		internalError(w, "could not load member", err)
		return false, false
	}
	if !canManage(actorRole, targetRole) {
		writeError(w, 403, "insufficient permissions")
		return false, false
	}
	return isInvitation, true
}

func (s *Server) updateMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "invalid member id")
		return
	}
	var changes map[string]string
	if !decodeJSON(w, r, &changes) {
		return
	}
	if role, ok := changes["role"]; ok && !isAssignableRole(role) {
		writeError(w, 400, "role must be admin or viewer")
		return
	}
	if status, ok := changes["status"]; ok && status != "active" && status != "revoked" {
		writeError(w, 400, "status must be active or revoked")
		return
	}
	isInvitation, ok := s.authorizeMemberChange(w, r, id)
	if !ok {
		return
	}
	if isInvitation {
		writeError(w, 400, "pending invitations cannot be modified; cancel and invite again")
		return
	}
	member, err := s.repository.UpdateMember(r.Context(), id, changes)
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "member not found")
			return
		}
		if err == repository.ErrNoChanges {
			writeError(w, 400, "nothing to update")
			return
		}
		internalError(w, "could not update member", err)
		return
	}
	writeJSON(w, 200, member)
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "invalid member id")
		return
	}
	isInvitation, ok := s.authorizeMemberChange(w, r, id)
	if !ok {
		return
	}
	remove := s.repository.DeleteMember
	if isInvitation {
		remove = s.repository.DeleteInvitation
	}
	if err := remove(r.Context(), id); err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "member not found")
			return
		}
		internalError(w, "could not delete member", err)
		return
	}
	w.WriteHeader(204)
}
