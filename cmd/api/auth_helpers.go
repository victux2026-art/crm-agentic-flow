package main

import (
	"encoding/json"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     int64  `json:"user_id"`
	TenantID   int64  `json:"tenant_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	TenantSlug string `json:"tenant_slug"`
	jwt.RegisteredClaims
}

type contextKey string

const authClaimsContextKey contextKey = "auth_claims"

func currentClaims(r *http.Request) (*Claims, bool) {
	claims, ok := r.Context().Value(authClaimsContextKey).(*Claims)
	return claims, ok
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
