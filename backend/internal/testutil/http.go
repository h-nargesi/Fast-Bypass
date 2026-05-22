package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func DoJSON(t *testing.T, h http.Handler, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func DecodeJSON(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(dst); err != nil {
		t.Fatalf("decode response %d: %v body=%s", w.Code, err, w.Body.String())
	}
}

func LoginToken(t *testing.T, h http.Handler, username, password string) string {
	t.Helper()
	w := DoJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username, "password": password,
	}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login %s: status=%d body=%s", username, w.Code, w.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	DecodeJSON(t, w, &resp)
	return resp.AccessToken
}
