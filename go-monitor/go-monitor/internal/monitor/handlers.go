package monitor

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Checker probes a URL and reports whether it is healthy.
type Checker func(url string) bool

// HTTPChecker treats any 2xx/3xx response as healthy.
func HTTPChecker(url string) bool {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}

// Server wires the store to HTTP routes.
type Server struct {
	store   *Store
	checker Checker
}

func NewServer(store *Store, checker Checker) *Server {
	return &Server{store: store, checker: checker}
}

// Routes returns the HTTP handler (Go 1.22 pattern routing).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /services", s.list)
	mux.HandleFunc("POST /services", s.create)
	mux.HandleFunc("GET /services/{name}", s.get)
	mux.HandleFunc("DELETE /services/{name}", s.remove)
	mux.HandleFunc("POST /services/{name}/check", s.check)
	return mux
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON"))
		return
	}
	svc, err := s.store.Add(in.Name, in.URL)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	svc, err := s.store.Get(r.PathValue("name"))
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.PathValue("name")); err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) check(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svc, err := s.store.Get(name)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	status := "down"
	if s.checker(svc.URL) {
		status = "up"
	}
	_ = s.store.SetStatus(name, status, time.Now().UTC())
	updated, _ := s.store.Get(name)
	writeJSON(w, http.StatusOK, updated)
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
