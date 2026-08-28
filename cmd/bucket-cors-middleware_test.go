// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio/internal/auth"
	"github.com/minio/minio/internal/bucket/cors"
)

func TestPerBucketCorsPreflight(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"http://example.com"},
		AllowedMethods: []string{"GET", "PUT"},
		AllowedHeaders: []string{"*"},
		ExposeHeaders:  []string{"ETag"},
		MaxAgeSeconds:  3000,
	}}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/mybucket/obj", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	handled := applyBucketCors(rec, req, cfg)
	if !handled {
		t.Fatal("expected preflight to be handled")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("allow-origin = %q", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("preflight status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "ETag" {
		t.Fatalf("expose-headers = %q", got)
	}
	requireCorsOriginVary(t, rec.Header())
}

func TestPerBucketCorsActualRequestNoMatchVariesByOrigin(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"https://allowed.example.com"},
		AllowedMethods: []string{"GET"},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mybucket/obj", nil)
	req.Header.Set("Origin", "https://denied.example.com")

	if handled := applyBucketCors(rec, req, cfg); handled {
		t.Fatal("actual request must continue when CORS does not match")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q", got)
	}
	requireCorsOriginVary(t, rec.Header())
}

func TestPerBucketCorsPreflightNoMatch(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"http://example.com"},
		AllowedMethods: []string{"GET"},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/mybucket/obj", nil)
	req.Header.Set("Origin", "http://evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	handled := applyBucketCors(rec, req, cfg)
	if !handled {
		t.Fatal("expected preflight to be handled (rejected)")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", rec.Code)
	}
	requireCorsVary(t, rec.Header())
}

func TestPerBucketCorsPreflightWildcardOrigin(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"*"},
		ExposeHeaders:  []string{"ETag"},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/mybucket/obj", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	if handled := applyBucketCors(rec, req, cfg); !handled {
		t.Fatal("expected preflight to be handled")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow-credentials = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "ETag" {
		t.Fatalf("expose-headers = %q", got)
	}
	requireCorsVary(t, rec.Header())
}

func TestPerBucketCorsActualRequest(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		ExposeHeaders:  []string{"ETag"},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mybucket/obj", nil)
	req.Header.Set("Origin", "http://any.com")

	handled := applyBucketCors(rec, req, cfg)
	if handled {
		t.Fatal("actual (non-preflight) request must not be terminated by CORS")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow-credentials = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "ETag" {
		t.Fatalf("expose-headers = %q", got)
	}
}

func TestPerBucketCorsOriginPatternResponse(t *testing.T) {
	cfg := &cors.Config{CORSRules: []cors.Rule{{
		AllowedOrigins: []string{"https://app.example.com", "https://*", "*"},
		AllowedMethods: []string{"GET"},
	}}}

	tests := []struct {
		origin          string
		wantOrigin      string
		wantCredentials string
	}{
		{"https://app.example.com", "https://app.example.com", "true"},
		{"https://other.example.com", "https://other.example.com", "true"},
		{"http://other.example.com", "*", ""},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/mybucket/obj", nil)
		req.Header.Set("Origin", tt.origin)
		if handled := applyBucketCors(rec, req, cfg); handled {
			t.Fatal("actual request must not be terminated by CORS")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
			t.Fatalf("origin %q: allow-origin = %q, want %q", tt.origin, got, tt.wantOrigin)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != tt.wantCredentials {
			t.Fatalf("origin %q: allow-credentials = %q, want %q", tt.origin, got, tt.wantCredentials)
		}
	}
}

func TestBucketCorsMetadataErrorFailsClosed(t *testing.T) {
	oldObjectAPI := newObjectLayerFn()
	setObjectLayer(nil)
	defer setObjectLayer(oldObjectAPI)

	wrapped := corsHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, getGetObjectURL("", "cors-metadata-error", "object"), nil)
		req.Header.Set("Origin", "https://app.example.com")
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want %d", method, rec.Code, http.StatusNoContent)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("%s metadata error fell back to global allow-origin %q", method, got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Fatalf("%s metadata error fell back to global credentials %q", method, got)
		}
	}
}

func TestBucketCorsNoConfigUsesGlobalFallback(t *testing.T) {
	ExecObjectLayerAPITest(ExecObjectLayerAPITestArgs{
		t:          t,
		objAPITest: testBucketCorsNoConfigUsesGlobalFallback,
		endpoints:  []string{"GetBucketCors"},
	})
}

func testBucketCorsNoConfigUsesGlobalFallback(_ ObjectLayer, _ string, bucket string, _ http.Handler, _ auth.Credentials, t *testing.T) {
	wrapped := corsHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, getGetObjectURL("", bucket, "object"), nil)
	req.Header.Set("Origin", "https://app.example.com")
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials = %q", got)
	}
}

func requireCorsVary(t *testing.T, header http.Header) {
	t.Helper()
	values := strings.Join(header.Values("Vary"), ",")
	for _, want := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !strings.Contains(values, want) {
			t.Fatalf("Vary = %q, missing %q", values, want)
		}
	}
}

func requireCorsOriginVary(t *testing.T, header http.Header) {
	t.Helper()
	if values := strings.Join(header.Values("Vary"), ","); !strings.Contains(values, "Origin") {
		t.Fatalf("Vary = %q, missing Origin", values)
	}
}
