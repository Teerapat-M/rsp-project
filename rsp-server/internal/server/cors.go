package server

import "strings"

// ParseOrigins splits a comma-separated CORS_ORIGINS value.
func ParseOrigins(csv string) []string {
	var out []string
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// OriginAllowed reports whether a browser Origin may talk to us.
// Empty origin (curl, smoke, no header) is always allowed.
// Empty allowlist means development: any origin.
func OriginAllowed(origin string, allow []string) bool {
	if origin == "" || len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if a == origin {
			return true
		}
	}
	return false
}
