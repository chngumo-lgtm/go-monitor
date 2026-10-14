// Package monitor holds a tiny in-memory service/health-check store.
package monitor

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("service not found")
	ErrDuplicate = errors.New("service already exists")
	ErrInvalid   = errors.New("name and url are required")
)

// Service is a monitored endpoint.
type Service struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Status      string    `json:"status"` // "unknown", "up" or "down"
	LastChecked time.Time `json:"last_checked,omitempty"`
}

// Store is a concurrency-safe in-memory registry.
type Store struct {
	mu       sync.RWMutex
	services map[string]*Service
}

func NewStore() *Store { return &Store{services: map[string]*Service{}} }

func (s *Store) Add(name, url string) (Service, error) {
	if name == "" || url == "" {
		return Service{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[name]; ok {
		return Service{}, ErrDuplicate
	}
	svc := &Service{Name: name, URL: url, Status: "unknown"}
	s.services[name] = svc
	return *svc, nil
}

func (s *Store) Get(name string) (Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svc, ok := s.services[name]
	if !ok {
		return Service{}, ErrNotFound
	}
	return *svc, nil
}

func (s *Store) List() []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Service, 0, len(s.services))
	for _, svc := range s.services {
		out = append(out, *svc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[name]; !ok {
		return ErrNotFound
	}
	delete(s.services, name)
	return nil
}

func (s *Store) SetStatus(name, status string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return ErrNotFound
	}
	svc.Status, svc.LastChecked = status, at
	return nil
}
