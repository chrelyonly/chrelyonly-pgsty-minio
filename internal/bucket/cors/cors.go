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

// Package cors implements the S3 per-bucket CORS configuration type,
// its validation, and origin/method/header matching helpers.
package cors

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/minio/pkg/v3/wildcard"
)

// maxCORSRules is the maximum number of rules allowed per bucket (AWS S3 limit).
const maxCORSRules = 100

// supportedMethods are the HTTP methods permitted in an AllowedMethod element.
var supportedMethods = map[string]bool{
	"GET":    true,
	"PUT":    true,
	"HEAD":   true,
	"POST":   true,
	"DELETE": true,
}

// Config is the S3 <CORSConfiguration> document.
type Config struct {
	XMLName   xml.Name `xml:"CORSConfiguration"`
	CORSRules []Rule   `xml:"CORSRule"`
}

// Rule is a single <CORSRule>.
type Rule struct {
	ID             string   `xml:"ID,omitempty"`
	AllowedHeaders []string `xml:"AllowedHeader"`
	AllowedMethods []string `xml:"AllowedMethod"`
	AllowedOrigins []string `xml:"AllowedOrigin"`
	ExposeHeaders  []string `xml:"ExposeHeader"`
	MaxAgeSeconds  int      `xml:"MaxAgeSeconds"`
}

// ParseBucketCorsConfig parses a CORS configuration from the given reader.
func ParseBucketCorsConfig(r io.Reader) (*Config, error) {
	var c Config
	if err := xml.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks the config against the S3 constraints.
func (c *Config) Validate() error {
	if len(c.CORSRules) == 0 {
		return errors.New("CORSConfiguration must contain at least one rule")
	}
	if len(c.CORSRules) > maxCORSRules {
		return errors.New("CORSConfiguration exceeds the maximum number of rules")
	}
	for _, r := range c.CORSRules {
		if len(r.AllowedOrigins) == 0 {
			return errors.New("CORSRule must contain at least one AllowedOrigin")
		}
		if len(r.AllowedMethods) == 0 {
			return errors.New("CORSRule must contain at least one AllowedMethod")
		}
		for _, m := range r.AllowedMethods {
			if !supportedMethods[strings.ToUpper(m)] {
				return errors.New("unsupported method in CORSRule: " + m)
			}
		}
		if r.MaxAgeSeconds < 0 {
			return errors.New("MaxAgeSeconds must not be negative")
		}
	}
	return nil
}

// HasAllowedOrigin reports whether the rule allows the given origin.
func (r Rule) HasAllowedOrigin(origin string) bool {
	for _, o := range r.AllowedOrigins {
		if o == "*" || wildcard.MatchSimple(o, origin) {
			return true
		}
	}
	return false
}

// HasAllowedMethod reports whether the rule allows the given HTTP method.
func (r Rule) HasAllowedMethod(method string) bool {
	for _, m := range r.AllowedMethods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// FilterAllowedHeaders returns the subset of reqHeaders permitted by the rule
// and whether every requested header was allowed.
func (r Rule) FilterAllowedHeaders(reqHeaders []string) ([]string, bool) {
	var allowed []string
	for _, h := range reqHeaders {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if !r.headerAllowed(h) {
			return nil, false
		}
		allowed = append(allowed, h)
	}
	return allowed, true
}

func (r Rule) headerAllowed(header string) bool {
	for _, h := range r.AllowedHeaders {
		if h == "*" || wildcard.MatchSimple(strings.ToLower(h), strings.ToLower(header)) {
			return true
		}
	}
	return false
}

// MatchRule returns the first rule whose origin and method both match.
func (c *Config) MatchRule(origin, method string) (*Rule, bool) {
	for i := range c.CORSRules {
		r := &c.CORSRules[i]
		if r.HasAllowedOrigin(origin) && r.HasAllowedMethod(method) {
			return r, true
		}
	}
	return nil, false
}
