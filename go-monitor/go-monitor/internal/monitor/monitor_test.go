package monitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(healthy bool) http.Handler {
	return NewServer(NewStore(), func(string) bool { return healthy }).Routes()
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStoreAddGetDelete(t *testing.T) {
	s := NewStore()
	if _, err := s.Add("api", "http://x"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("api", "http://x"); err != ErrDuplicate {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	if _, err := s.Add("", ""); err != ErrInvalid {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if err := s.Delete("api"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("api"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestStoreListSorted(t *testing.T) {
	s := NewStore()
	s.Add("b", "u")
	s.Add("a", "u")
	got := s.List()
	if len(got) != 2 || got[0].Name != "a" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestSetStatus(t *testing.T) {
	s := NewStore()
	s.Add("a", "u")
	if err := s.SetStatus("a", "up", time.Now()); err != nil {
		t.Fatal(err)
	}
	if svc, _ := s.Get("a"); svc.Status != "up" {
		t.Fatalf("status not updated: %+v", svc)
	}
	if err := s.SetStatus("zzz", "up", time.Now()); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestHealthz(t *testing.T) {
	if rec := do(newTestServer(true), "GET", "/healthz", ""); rec.Code != 200 {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestCreateAndGetEndpoint(t *testing.T) {
	h := newTestServer(true)
	if rec := do(h, "POST", "/services", `{"name":"web","url":"http://example.com"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d", rec.Code)
	}
	if rec := do(h, "POST", "/services", `{"name":"web","url":"http://example.com"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: got %d", rec.Code)
	}
	if rec := do(h, "POST", "/services", `not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json: got %d", rec.Code)
	}
	if rec := do(h, "POST", "/services", `{"name":"","url":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid: got %d", rec.Code)
	}
	if rec := do(h, "GET", "/services/web", ""); rec.Code != 200 {
		t.Fatalf("get: got %d", rec.Code)
	}
	if rec := do(h, "GET", "/services/nope", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: got %d", rec.Code)
	}
}

func TestCheckEndpoint(t *testing.T) {
	for _, tc := range []struct {
		healthy bool
		want    string
	}{{true, `"status":"up"`}, {false, `"status":"down"`}} {
		h := newTestServer(tc.healthy)
		do(h, "POST", "/services", `{"name":"web","url":"http://example.com"}`)
		rec := do(h, "POST", "/services/web/check", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("healthy=%v: code=%d body=%s", tc.healthy, rec.Code, rec.Body.String())
		}
	}
	if rec := do(newTestServer(true), "POST", "/services/none/check", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	h := newTestServer(true)
	do(h, "POST", "/services", `{"name":"web","url":"u"}`)
	if rec := do(h, "DELETE", "/services/web", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("got %d", rec.Code)
	}
	if rec := do(h, "DELETE", "/services/web", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestList(t *testing.T) {
	h := newTestServer(true)
	do(h, "POST", "/services", `{"name":"web","url":"u"}`)
	rec := do(h, "GET", "/services", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"web"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}
