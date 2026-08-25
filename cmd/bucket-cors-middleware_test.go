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
	"testing"

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
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://any.com" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "ETag" {
		t.Fatalf("expose-headers = %q", got)
	}
}
