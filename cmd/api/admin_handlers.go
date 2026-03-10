package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	chi "github.com/go-chi/chi/v5"
	pgx "github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func getAdminTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAdminClaims(w, r)
	if !ok {
		return
	}

	var tenant TenantAdminProfile
	err := dbPool.QueryRow(r.Context(), `
		SELECT id, name, slug, plan, status, created_at, updated_at
		FROM tenants
		WHERE id = $1`, claims.TenantID,
	).Scan(&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Plan, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error loading admin tenant: %v\n", err)
		http.Error(w, "Could not fetch tenant", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tenant)
}

func updateAdminTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAdminClaims(w, r)
	if !ok {
		return
	}

	var tenant TenantAdminProfile
	if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if tenant.Name == "" || tenant.Slug == "" {
		http.Error(w, "Tenant name and slug are required", http.StatusBadRequest)
		return
	}

	err := dbPool.QueryRow(r.Context(), `
		UPDATE tenants
		SET name = $1, slug = $2, plan = $3, status = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, name, slug, plan, status, created_at, updated_at`,
		tenant.Name, tenant.Slug, tenant.Plan, tenant.Status, claims.TenantID,
	).Scan(&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Plan, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating tenant: %v\n", err)
		http.Error(w, "Could not update tenant", http.StatusInternalServerError)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "updated", "tenant", claims.TenantID, tenant)
	writeJSON(w, http.StatusOK, tenant)
}

func getAdminUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAdminClaims(w, r)
	if !ok {
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT u.id, u.email, u.full_name, u.status, tm.role, tm.status, u.last_login_at, u.created_at, u.updated_at
		FROM tenant_memberships tm
		INNER JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id = $1
		ORDER BY u.created_at ASC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying admin users: %v\n", err)
		http.Error(w, "Could not fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&user.UserStatus,
			&user.Role,
			&user.MembershipStatus,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning admin user row: %v\n", err)
			http.Error(w, "Could not fetch users", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	writeJSON(w, http.StatusOK, users)
}

func createAdminUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAdminClaims(w, r)
	if !ok {
		return
	}

	var input struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		FullName         string `json:"full_name"`
		UserStatus       string `json:"user_status"`
		Role             string `json:"role"`
		MembershipStatus string `json:"membership_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if input.Email == "" || input.Password == "" || input.FullName == "" {
		http.Error(w, "Email, password, and full_name are required", http.StatusBadRequest)
		return
	}
	if input.UserStatus == "" {
		input.UserStatus = "active"
	}
	if input.Role == "" {
		input.Role = "member"
	}
	if input.MembershipStatus == "" {
		input.MembershipStatus = "active"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing user password: %v\n", err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	tx, err := dbPool.Begin(r.Context())
	if err != nil {
		log.Printf("Error opening create user transaction: %v\n", err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	var user AdminUser
	err = tx.QueryRow(r.Context(), `
		INSERT INTO users (email, password_hash, full_name, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, full_name, status, last_login_at, created_at, updated_at`,
		input.Email, string(hash), input.FullName, input.UserStatus,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.UserStatus, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting admin user: %v\n", err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(r.Context(), `
		INSERT INTO tenant_memberships (tenant_id, user_id, role, status)
		VALUES ($1, $2, $3, $4)`,
		claims.TenantID, user.ID, input.Role, input.MembershipStatus,
	); err != nil {
		log.Printf("Error inserting tenant membership: %v\n", err)
		http.Error(w, "Could not create user membership", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		log.Printf("Error committing create user transaction: %v\n", err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	user.Role = input.Role
	user.MembershipStatus = input.MembershipStatus
	writeAuditLog(r.Context(), r, claims.TenantID, "created", "user", user.ID, user)
	writeJSON(w, http.StatusCreated, user)
}

func updateAdminUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAdminClaims(w, r)
	if !ok {
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var input struct {
		FullName         string `json:"full_name"`
		UserStatus       string `json:"user_status"`
		Role             string `json:"role"`
		MembershipStatus string `json:"membership_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if input.FullName == "" {
		http.Error(w, "full_name is required", http.StatusBadRequest)
		return
	}
	if input.UserStatus == "" {
		input.UserStatus = "active"
	}
	if input.Role == "" {
		input.Role = "member"
	}
	if input.MembershipStatus == "" {
		input.MembershipStatus = "active"
	}

	tx, err := dbPool.Begin(r.Context())
	if err != nil {
		log.Printf("Error opening update user transaction: %v\n", err)
		http.Error(w, "Could not update user", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	var user AdminUser
	err = tx.QueryRow(r.Context(), `
		UPDATE users
		SET full_name = $1, status = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, full_name, status, last_login_at, created_at, updated_at`,
		input.FullName, input.UserStatus, userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.UserStatus, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating admin user: %v\n", err)
		http.Error(w, "Could not update user", http.StatusInternalServerError)
		return
	}

	commandTag, err := tx.Exec(r.Context(), `
		UPDATE tenant_memberships
		SET role = $1, status = $2
		WHERE tenant_id = $3 AND user_id = $4`,
		input.Role, input.MembershipStatus, claims.TenantID, userID,
	)
	if err != nil {
		log.Printf("Error updating tenant membership: %v\n", err)
		http.Error(w, "Could not update user membership", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "User membership not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		log.Printf("Error committing update user transaction: %v\n", err)
		http.Error(w, "Could not update user", http.StatusInternalServerError)
		return
	}

	user.Role = input.Role
	user.MembershipStatus = input.MembershipStatus
	writeAuditLog(r.Context(), r, claims.TenantID, "updated", "user", user.ID, user)
	writeJSON(w, http.StatusOK, user)
}
