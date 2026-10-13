package api

import (
	"net/http"
	"strconv"
	"strings"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/repository"
)

func (s *Server) listAddresses(w http.ResponseWriter, r *http.Request) {
	addresses, err := s.repository.ListAddresses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list addresses")
		return
	}
	writeJSON(w, http.StatusOK, addresses)
}

func (s *Server) createAddress(w http.ResponseWriter, r *http.Request) {
	var input models.AddressRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Value) == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}
	address, err := s.repository.CreateAddress(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create address")
		return
	}
	writeJSON(w, http.StatusCreated, address)
}

func (s *Server) deleteAddress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid address id")
		return
	}
	if err := s.repository.DeleteAddress(r.Context(), id); err != nil {
		if repository.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "address not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
