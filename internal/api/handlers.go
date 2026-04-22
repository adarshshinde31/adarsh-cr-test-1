package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/adarshshinde/feature-flags/internal/flags"
)

type Server struct {
	store flags.Store
}

func NewServer(store flags.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /flags", s.list)
	mux.HandleFunc("POST /flags", s.create)
	mux.HandleFunc("GET /flags/{key}", s.get)
	mux.HandleFunc("PUT /flags/{key}", s.update)
	mux.HandleFunc("DELETE /flags/{key}", s.delete)
	return mux
}

type flagPayload struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var body flagPayload
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	flag := flags.FeatureFlag{
		Key:         body.Key,
		Enabled:     body.Enabled,
		Description: body.Description,
	}

	if err := s.store.Create(r.Context(), flag); err != nil {
		switch {
		case errors.Is(err, flags.ErrExists):
			writeError(w, http.StatusConflict, err)
		case errors.Is(err, flags.ErrInvalid):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	created, err := s.store.Get(r.Context(), flag.Key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	flag, err := s.store.Get(r.Context(), key)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var body flagPayload
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Path key wins — ignore body.Key to avoid accidental renames.
	flag := flags.FeatureFlag{
		Key:         key,
		Enabled:     body.Enabled,
		Description: body.Description,
	}

	if err := s.store.Update(r.Context(), flag); err != nil {
		s.writeStoreError(w, err)
		return
	}

	updated, err := s.store.Get(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := s.store.Delete(r.Context(), key); err != nil {
		s.writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, flags.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, flags.ErrInvalid):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
