package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	chi "github.com/go-chi/chi/v5"
	pgx "github.com/jackc/pgx/v5"
)

func createOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if org.Name == "" {
		http.Error(w, "Organization name is required", http.StatusBadRequest)
		return
	}

	org.TenantID = claims.TenantID
	if org.Metadata == nil {
		org.Metadata = map[string]interface{}{}
	}
	if org.OwnerUserID == nil {
		org.OwnerUserID = &claims.UserID
	}

	query := `
		INSERT INTO organizations (tenant_id, name, domain, industry, size_band, country, owner_user_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(r.Context(), query,
		org.TenantID, org.Name, org.Domain, org.Industry, org.SizeBand, org.Country, org.OwnerUserID, org.Metadata,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting organization: %v\n", err)
		http.Error(w, "Could not create organization", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &org.TenantID, "organization.created", org)
	writeAuditLog(r.Context(), r, org.TenantID, "created", "organization", org.ID, org)
	writeJSON(w, http.StatusCreated, org)
}

func getOrganizations(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, name, domain, industry, size_band, country, owner_user_id, metadata, created_at, updated_at
		FROM organizations
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying organizations: %v\n", err)
		http.Error(w, "Could not fetch organizations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var organizations []Organization
	for rows.Next() {
		var org Organization
		err := rows.Scan(
			&org.ID, &org.TenantID, &org.Name, &org.Domain, &org.Industry, &org.SizeBand,
			&org.Country, &org.OwnerUserID, &org.Metadata, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning organization row: %v\n", err)
			http.Error(w, "Could not fetch organizations", http.StatusInternalServerError)
			return
		}
		organizations = append(organizations, org)
	}

	writeJSON(w, http.StatusOK, organizations)
}

func getOrganizationByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var org Organization
	query := `
		SELECT id, tenant_id, name, domain, industry, size_band, country, owner_user_id, metadata, created_at, updated_at
		FROM organizations
		WHERE id = $1 AND tenant_id = $2`

	err = dbPool.QueryRow(r.Context(), query, id, claims.TenantID).Scan(
		&org.ID, &org.TenantID, &org.Name, &org.Domain, &org.Industry, &org.SizeBand,
		&org.Country, &org.OwnerUserID, &org.Metadata, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying organization by ID: %v\n", err)
		http.Error(w, "Could not fetch organization", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, org)
}

func updateOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if org.Name == "" {
		http.Error(w, "Organization name is required", http.StatusBadRequest)
		return
	}
	if org.Metadata == nil {
		org.Metadata = map[string]interface{}{}
	}
	if org.OwnerUserID == nil {
		org.OwnerUserID = &claims.UserID
	}

	org.ID = id
	org.TenantID = claims.TenantID
	err = dbPool.QueryRow(r.Context(), `
		UPDATE organizations
		SET name = $1, domain = $2, industry = $3, size_band = $4, country = $5,
		    owner_user_id = $6, metadata = $7, updated_at = NOW()
		WHERE id = $8 AND tenant_id = $9
		RETURNING created_at, updated_at`,
		org.Name, org.Domain, org.Industry, org.SizeBand, org.Country, org.OwnerUserID, org.Metadata, org.ID, org.TenantID,
	).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating organization: %v\n", err)
		http.Error(w, "Could not update organization", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &org.TenantID, "organization.updated", org)
	writeAuditLog(r.Context(), r, org.TenantID, "updated", "organization", org.ID, org)
	writeJSON(w, http.StatusOK, org)
}

func deleteOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM organizations
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting organization: %v\n", err)
		http.Error(w, "Could not delete organization", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "deleted", "organization", id, map[string]interface{}{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func createMVPPerson(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var person MVPPerson
	if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if person.FirstName == "" || person.LastName == "" {
		http.Error(w, "First name and last name are required", http.StatusBadRequest)
		return
	}

	person.TenantID = claims.TenantID
	if person.Status == "" {
		person.Status = "active"
	}
	if person.Metadata == nil {
		person.Metadata = map[string]interface{}{}
	}
	if person.OwnerUserID == nil {
		person.OwnerUserID = &claims.UserID
	}

	query := `
		INSERT INTO people (
			tenant_id, organization_id, first_name, last_name, email, phone, job_title,
			linkedin_url, status, owner_user_id, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(r.Context(), query,
		person.TenantID, person.OrganizationID, person.FirstName, person.LastName, person.Email, person.Phone,
		person.JobTitle, person.LinkedinURL, person.Status, person.OwnerUserID, person.Metadata,
	).Scan(&person.ID, &person.CreatedAt, &person.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting MVP person: %v\n", err)
		http.Error(w, "Could not create person", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &person.TenantID, "person.created", person)
	writeAuditLog(r.Context(), r, person.TenantID, "created", "person", person.ID, person)
	writeJSON(w, http.StatusCreated, person)
}

func getMVPPeople(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, organization_id, first_name, last_name, email, phone, job_title,
		       linkedin_url, status, owner_user_id, metadata, created_at, updated_at
		FROM people
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying MVP people: %v\n", err)
		http.Error(w, "Could not fetch people", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var people []MVPPerson
	for rows.Next() {
		var person MVPPerson
		err := rows.Scan(
			&person.ID, &person.TenantID, &person.OrganizationID, &person.FirstName, &person.LastName,
			&person.Email, &person.Phone, &person.JobTitle, &person.LinkedinURL, &person.Status,
			&person.OwnerUserID, &person.Metadata, &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning MVP person row: %v\n", err)
			http.Error(w, "Could not fetch people", http.StatusInternalServerError)
			return
		}
		people = append(people, person)
	}

	writeJSON(w, http.StatusOK, people)
}

func getMVPPersonByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	var person MVPPerson
	query := `
		SELECT id, tenant_id, organization_id, first_name, last_name, email, phone, job_title,
		       linkedin_url, status, owner_user_id, metadata, created_at, updated_at
		FROM people
		WHERE id = $1 AND tenant_id = $2`

	err = dbPool.QueryRow(r.Context(), query, id, claims.TenantID).Scan(
		&person.ID, &person.TenantID, &person.OrganizationID, &person.FirstName, &person.LastName,
		&person.Email, &person.Phone, &person.JobTitle, &person.LinkedinURL, &person.Status,
		&person.OwnerUserID, &person.Metadata, &person.CreatedAt, &person.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying MVP person by ID: %v\n", err)
		http.Error(w, "Could not fetch person", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, person)
}

func updateMVPPerson(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	var person MVPPerson
	if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if person.FirstName == "" || person.LastName == "" {
		http.Error(w, "First name and last name are required", http.StatusBadRequest)
		return
	}
	if person.Status == "" {
		person.Status = "active"
	}
	if person.Metadata == nil {
		person.Metadata = map[string]interface{}{}
	}
	if person.OwnerUserID == nil {
		person.OwnerUserID = &claims.UserID
	}

	person.ID = id
	person.TenantID = claims.TenantID
	err = dbPool.QueryRow(r.Context(), `
		UPDATE people
		SET organization_id = $1, first_name = $2, last_name = $3, email = $4, phone = $5,
		    job_title = $6, linkedin_url = $7, status = $8, owner_user_id = $9, metadata = $10, updated_at = NOW()
		WHERE id = $11 AND tenant_id = $12
		RETURNING created_at, updated_at`,
		person.OrganizationID, person.FirstName, person.LastName, person.Email, person.Phone, person.JobTitle,
		person.LinkedinURL, person.Status, person.OwnerUserID, person.Metadata, person.ID, person.TenantID,
	).Scan(&person.CreatedAt, &person.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating MVP person: %v\n", err)
		http.Error(w, "Could not update person", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &person.TenantID, "person.updated", person)
	writeAuditLog(r.Context(), r, person.TenantID, "updated", "person", person.ID, person)
	writeJSON(w, http.StatusOK, person)
}

func deleteMVPPerson(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM people
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting MVP person: %v\n", err)
		http.Error(w, "Could not delete person", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "deleted", "person", id, map[string]interface{}{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func createDeal(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var deal Deal
	if err := json.NewDecoder(r.Body).Decode(&deal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if deal.Name == "" {
		http.Error(w, "Deal name is required", http.StatusBadRequest)
		return
	}
	if deal.Stage == "" {
		deal.Stage = "lead"
	}
	if deal.Status == "" {
		deal.Status = "open"
	}
	if deal.ValueCurrency == "" {
		deal.ValueCurrency = "USD"
	}
	if deal.Metadata == nil {
		deal.Metadata = map[string]interface{}{}
	}
	deal.TenantID = claims.TenantID
	if deal.OwnerUserID == nil {
		deal.OwnerUserID = &claims.UserID
	}

	query := `
		INSERT INTO deals (
			tenant_id, organization_id, primary_person_id, name, stage, status, value_amount,
			value_currency, close_date_expected, close_date_actual, owner_user_id, health_score, source, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(r.Context(), query,
		deal.TenantID, deal.OrganizationID, deal.PrimaryPersonID, deal.Name, deal.Stage, deal.Status,
		deal.ValueAmount, deal.ValueCurrency, deal.CloseDateExpected, deal.CloseDateActual, deal.OwnerUserID,
		deal.HealthScore, deal.Source, deal.Metadata,
	).Scan(&deal.ID, &deal.CreatedAt, &deal.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting deal: %v\n", err)
		http.Error(w, "Could not create deal", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &deal.TenantID, "deal.created", deal)
	writeAuditLog(r.Context(), r, deal.TenantID, "created", "deal", deal.ID, deal)
	writeJSON(w, http.StatusCreated, deal)
}

func getDeals(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, organization_id, primary_person_id, name, stage, status, value_amount,
		       value_currency, close_date_expected, close_date_actual, owner_user_id, health_score, source,
		       metadata, created_at, updated_at
		FROM deals
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying deals: %v\n", err)
		http.Error(w, "Could not fetch deals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var deals []Deal
	for rows.Next() {
		var deal Deal
		err := rows.Scan(
			&deal.ID, &deal.TenantID, &deal.OrganizationID, &deal.PrimaryPersonID, &deal.Name, &deal.Stage,
			&deal.Status, &deal.ValueAmount, &deal.ValueCurrency, &deal.CloseDateExpected, &deal.CloseDateActual,
			&deal.OwnerUserID, &deal.HealthScore, &deal.Source, &deal.Metadata, &deal.CreatedAt, &deal.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning deal row: %v\n", err)
			http.Error(w, "Could not fetch deals", http.StatusInternalServerError)
			return
		}
		deals = append(deals, deal)
	}

	writeJSON(w, http.StatusOK, deals)
}

func getDealByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid deal ID", http.StatusBadRequest)
		return
	}

	var deal Deal
	query := `
		SELECT id, tenant_id, organization_id, primary_person_id, name, stage, status, value_amount,
		       value_currency, close_date_expected, close_date_actual, owner_user_id, health_score, source,
		       metadata, created_at, updated_at
		FROM deals
		WHERE id = $1 AND tenant_id = $2`

	err = dbPool.QueryRow(r.Context(), query, id, claims.TenantID).Scan(
		&deal.ID, &deal.TenantID, &deal.OrganizationID, &deal.PrimaryPersonID, &deal.Name, &deal.Stage,
		&deal.Status, &deal.ValueAmount, &deal.ValueCurrency, &deal.CloseDateExpected, &deal.CloseDateActual,
		&deal.OwnerUserID, &deal.HealthScore, &deal.Source, &deal.Metadata, &deal.CreatedAt, &deal.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Deal not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying deal by ID: %v\n", err)
		http.Error(w, "Could not fetch deal", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, deal)
}

func updateDeal(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid deal ID", http.StatusBadRequest)
		return
	}

	var deal Deal
	if err := json.NewDecoder(r.Body).Decode(&deal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if deal.Name == "" {
		http.Error(w, "Deal name is required", http.StatusBadRequest)
		return
	}
	if deal.Stage == "" {
		deal.Stage = "lead"
	}
	if deal.Status == "" {
		deal.Status = "open"
	}
	if deal.ValueCurrency == "" {
		deal.ValueCurrency = "USD"
	}
	if deal.Metadata == nil {
		deal.Metadata = map[string]interface{}{}
	}
	if deal.OwnerUserID == nil {
		deal.OwnerUserID = &claims.UserID
	}

	deal.ID = id
	deal.TenantID = claims.TenantID
	err = dbPool.QueryRow(r.Context(), `
		UPDATE deals
		SET organization_id = $1, primary_person_id = $2, name = $3, stage = $4, status = $5,
		    value_amount = $6, value_currency = $7, close_date_expected = $8, close_date_actual = $9,
		    owner_user_id = $10, health_score = $11, source = $12, metadata = $13, updated_at = NOW()
		WHERE id = $14 AND tenant_id = $15
		RETURNING created_at, updated_at`,
		deal.OrganizationID, deal.PrimaryPersonID, deal.Name, deal.Stage, deal.Status, deal.ValueAmount,
		deal.ValueCurrency, deal.CloseDateExpected, deal.CloseDateActual, deal.OwnerUserID, deal.HealthScore,
		deal.Source, deal.Metadata, deal.ID, deal.TenantID,
	).Scan(&deal.CreatedAt, &deal.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Deal not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating deal: %v\n", err)
		http.Error(w, "Could not update deal", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &deal.TenantID, "deal.updated", deal)
	writeAuditLog(r.Context(), r, deal.TenantID, "updated", "deal", deal.ID, deal)
	writeJSON(w, http.StatusOK, deal)
}

func deleteDeal(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid deal ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM deals
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting deal: %v\n", err)
		http.Error(w, "Could not delete deal", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Deal not found", http.StatusNotFound)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "deleted", "deal", id, map[string]interface{}{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, "Task title is required", http.StatusBadRequest)
		return
	}

	task.TenantID = claims.TenantID
	if task.Status == "" {
		task.Status = "open"
	}
	if task.Priority == "" {
		task.Priority = "normal"
	}
	if task.Source == "" {
		task.Source = "manual"
	}
	if task.OwnerUserID == nil {
		task.OwnerUserID = &claims.UserID
	}
	if task.CreatedByUserID == nil {
		task.CreatedByUserID = &claims.UserID
	}

	query := `
		INSERT INTO tasks (
			tenant_id, deal_id, person_id, organization_id, title, description, status,
			priority, due_at, owner_user_id, created_by_user_id, source
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(r.Context(), query,
		task.TenantID, task.DealID, task.PersonID, task.OrganizationID, task.Title, task.Description,
		task.Status, task.Priority, task.DueAt, task.OwnerUserID, task.CreatedByUserID, task.Source,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting task: %v\n", err)
		http.Error(w, "Could not create task", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &task.TenantID, "task.created", task)
	writeAuditLog(r.Context(), r, task.TenantID, "created", "task", task.ID, task)
	writeJSON(w, http.StatusCreated, task)
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, deal_id, person_id, organization_id, title, description, status,
		       priority, due_at, owner_user_id, created_by_user_id, source, created_at, updated_at
		FROM tasks
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying tasks: %v\n", err)
		http.Error(w, "Could not fetch tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(
			&task.ID, &task.TenantID, &task.DealID, &task.PersonID, &task.OrganizationID, &task.Title,
			&task.Description, &task.Status, &task.Priority, &task.DueAt, &task.OwnerUserID,
			&task.CreatedByUserID, &task.Source, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning task row: %v\n", err)
			http.Error(w, "Could not fetch tasks", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	writeJSON(w, http.StatusOK, tasks)
}

func getTaskByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task Task
	query := `
		SELECT id, tenant_id, deal_id, person_id, organization_id, title, description, status,
		       priority, due_at, owner_user_id, created_by_user_id, source, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND tenant_id = $2`

	err = dbPool.QueryRow(r.Context(), query, id, claims.TenantID).Scan(
		&task.ID, &task.TenantID, &task.DealID, &task.PersonID, &task.OrganizationID, &task.Title,
		&task.Description, &task.Status, &task.Priority, &task.DueAt, &task.OwnerUserID,
		&task.CreatedByUserID, &task.Source, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying task by ID: %v\n", err)
		http.Error(w, "Could not fetch task", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, "Task title is required", http.StatusBadRequest)
		return
	}

	task.ID = id
	task.TenantID = claims.TenantID
	if task.Status == "" {
		task.Status = "open"
	}
	if task.Priority == "" {
		task.Priority = "normal"
	}
	if task.Source == "" {
		task.Source = "manual"
	}
	if task.OwnerUserID == nil {
		task.OwnerUserID = &claims.UserID
	}
	if task.CreatedByUserID == nil {
		task.CreatedByUserID = &claims.UserID
	}

	err = dbPool.QueryRow(r.Context(), `
		UPDATE tasks
		SET deal_id = $1, person_id = $2, organization_id = $3, title = $4, description = $5,
		    status = $6, priority = $7, due_at = $8, owner_user_id = $9, created_by_user_id = $10,
		    source = $11, updated_at = NOW()
		WHERE id = $12 AND tenant_id = $13
		RETURNING created_at, updated_at`,
		task.DealID, task.PersonID, task.OrganizationID, task.Title, task.Description, task.Status,
		task.Priority, task.DueAt, task.OwnerUserID, task.CreatedByUserID, task.Source, task.ID, task.TenantID,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating task: %v\n", err)
		http.Error(w, "Could not update task", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &task.TenantID, "task.updated", task)
	writeAuditLog(r.Context(), r, task.TenantID, "updated", "task", task.ID, task)
	writeJSON(w, http.StatusOK, task)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM tasks
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting task: %v\n", err)
		http.Error(w, "Could not delete task", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "deleted", "task", id, map[string]interface{}{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if note.Body == "" {
		http.Error(w, "Note body is required", http.StatusBadRequest)
		return
	}

	note.TenantID = claims.TenantID
	if note.Source == "" {
		note.Source = "manual"
	}
	if note.AuthorUserID == nil {
		note.AuthorUserID = &claims.UserID
	}

	query := `
		INSERT INTO notes (tenant_id, deal_id, person_id, organization_id, author_user_id, body, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(r.Context(), query,
		note.TenantID, note.DealID, note.PersonID, note.OrganizationID, note.AuthorUserID, note.Body, note.Source,
	).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting note: %v\n", err)
		http.Error(w, "Could not create note", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &note.TenantID, "note.created", note)
	writeAuditLog(r.Context(), r, note.TenantID, "created", "note", note.ID, note)
	writeJSON(w, http.StatusCreated, note)
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, deal_id, person_id, organization_id, author_user_id, body, source, created_at, updated_at
		FROM notes
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying notes: %v\n", err)
		http.Error(w, "Could not fetch notes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(
			&note.ID, &note.TenantID, &note.DealID, &note.PersonID, &note.OrganizationID,
			&note.AuthorUserID, &note.Body, &note.Source, &note.CreatedAt, &note.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning note row: %v\n", err)
			http.Error(w, "Could not fetch notes", http.StatusInternalServerError)
			return
		}
		notes = append(notes, note)
	}

	writeJSON(w, http.StatusOK, notes)
}

func getNoteByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	var note Note
	query := `
		SELECT id, tenant_id, deal_id, person_id, organization_id, author_user_id, body, source, created_at, updated_at
		FROM notes
		WHERE id = $1 AND tenant_id = $2`

	err = dbPool.QueryRow(r.Context(), query, id, claims.TenantID).Scan(
		&note.ID, &note.TenantID, &note.DealID, &note.PersonID, &note.OrganizationID,
		&note.AuthorUserID, &note.Body, &note.Source, &note.CreatedAt, &note.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying note by ID: %v\n", err)
		http.Error(w, "Could not fetch note", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, note)
}

func updateNote(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if note.Body == "" {
		http.Error(w, "Note body is required", http.StatusBadRequest)
		return
	}

	note.ID = id
	note.TenantID = claims.TenantID
	if note.Source == "" {
		note.Source = "manual"
	}
	if note.AuthorUserID == nil {
		note.AuthorUserID = &claims.UserID
	}

	err = dbPool.QueryRow(r.Context(), `
		UPDATE notes
		SET deal_id = $1, person_id = $2, organization_id = $3, author_user_id = $4,
		    body = $5, source = $6, updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
		RETURNING created_at, updated_at`,
		note.DealID, note.PersonID, note.OrganizationID, note.AuthorUserID, note.Body,
		note.Source, note.ID, note.TenantID,
	).Scan(&note.CreatedAt, &note.UpdatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating note: %v\n", err)
		http.Error(w, "Could not update note", http.StatusInternalServerError)
		return
	}

	enqueueOutboxEvent(r.Context(), &note.TenantID, "note.updated", note)
	writeAuditLog(r.Context(), r, note.TenantID, "updated", "note", note.ID, note)
	writeJSON(w, http.StatusOK, note)
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM notes
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting note: %v\n", err)
		http.Error(w, "Could not delete note", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	writeAuditLog(r.Context(), r, claims.TenantID, "deleted", "note", id, map[string]interface{}{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func createWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var endpoint WebhookEndpoint
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if endpoint.Name == "" || endpoint.TargetURL == "" || endpoint.SigningSecret == "" {
		http.Error(w, "Name, target_url and signing_secret are required", http.StatusBadRequest)
		return
	}

	endpoint.TenantID = claims.TenantID
	if endpoint.Status == "" {
		endpoint.Status = "active"
	}
	if endpoint.CreatedByUserID == nil {
		endpoint.CreatedByUserID = &claims.UserID
	}

	err := dbPool.QueryRow(r.Context(), `
		INSERT INTO webhook_endpoints (tenant_id, name, target_url, signing_secret, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		endpoint.TenantID, endpoint.Name, endpoint.TargetURL, endpoint.SigningSecret, endpoint.Status, endpoint.CreatedByUserID,
	).Scan(&endpoint.ID, &endpoint.CreatedAt)
	if err != nil {
		log.Printf("Error inserting webhook endpoint: %v\n", err)
		http.Error(w, "Could not create webhook endpoint", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, endpoint)
}

func getWebhookEndpoints(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, name, target_url, signing_secret, status, created_by_user_id, created_at
		FROM webhook_endpoints
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying webhook endpoints: %v\n", err)
		http.Error(w, "Could not fetch webhook endpoints", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var endpoints []WebhookEndpoint
	for rows.Next() {
		var endpoint WebhookEndpoint
		if err := rows.Scan(
			&endpoint.ID, &endpoint.TenantID, &endpoint.Name, &endpoint.TargetURL, &endpoint.SigningSecret,
			&endpoint.Status, &endpoint.CreatedByUserID, &endpoint.CreatedAt,
		); err != nil {
			log.Printf("Error scanning webhook endpoint row: %v\n", err)
			http.Error(w, "Could not fetch webhook endpoints", http.StatusInternalServerError)
			return
		}
		endpoints = append(endpoints, endpoint)
	}

	writeJSON(w, http.StatusOK, endpoints)
}

func updateWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook endpoint ID", http.StatusBadRequest)
		return
	}

	var endpoint WebhookEndpoint
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if endpoint.Name == "" || endpoint.TargetURL == "" || endpoint.SigningSecret == "" {
		http.Error(w, "Name, target_url and signing_secret are required", http.StatusBadRequest)
		return
	}
	if endpoint.Status == "" {
		endpoint.Status = "active"
	}

	endpoint.ID = id
	endpoint.TenantID = claims.TenantID
	err = dbPool.QueryRow(r.Context(), `
		UPDATE webhook_endpoints
		SET name = $1, target_url = $2, signing_secret = $3, status = $4
		WHERE id = $5 AND tenant_id = $6
		RETURNING created_by_user_id, created_at`,
		endpoint.Name, endpoint.TargetURL, endpoint.SigningSecret, endpoint.Status, endpoint.ID, endpoint.TenantID,
	).Scan(&endpoint.CreatedByUserID, &endpoint.CreatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Webhook endpoint not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating webhook endpoint: %v\n", err)
		http.Error(w, "Could not update webhook endpoint", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

func deleteWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook endpoint ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM webhook_endpoints
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting webhook endpoint: %v\n", err)
		http.Error(w, "Could not delete webhook endpoint", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Webhook endpoint not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var subscription WebhookSubscription
	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if subscription.WebhookEndpointID == 0 || subscription.EventType == "" {
		http.Error(w, "webhook_endpoint_id and event_type are required", http.StatusBadRequest)
		return
	}

	subscription.TenantID = claims.TenantID
	subscription.IsActive = true

	err := dbPool.QueryRow(r.Context(), `
		INSERT INTO webhook_subscriptions (tenant_id, webhook_endpoint_id, event_type, is_active)
		SELECT $1, we.id, $2, $3
		FROM webhook_endpoints we
		WHERE we.id = $4 AND we.tenant_id = $1
		RETURNING id, created_at`,
		subscription.TenantID, subscription.EventType, subscription.IsActive, subscription.WebhookEndpointID,
	).Scan(&subscription.ID, &subscription.CreatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Webhook endpoint not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error inserting webhook subscription: %v\n", err)
		http.Error(w, "Could not create webhook subscription", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, subscription)
}

func getWebhookSubscriptions(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := dbPool.Query(r.Context(), `
		SELECT id, tenant_id, webhook_endpoint_id, event_type, is_active, created_at
		FROM webhook_subscriptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying webhook subscriptions: %v\n", err)
		http.Error(w, "Could not fetch webhook subscriptions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var subscriptions []WebhookSubscription
	for rows.Next() {
		var subscription WebhookSubscription
		if err := rows.Scan(
			&subscription.ID, &subscription.TenantID, &subscription.WebhookEndpointID,
			&subscription.EventType, &subscription.IsActive, &subscription.CreatedAt,
		); err != nil {
			log.Printf("Error scanning webhook subscription row: %v\n", err)
			http.Error(w, "Could not fetch webhook subscriptions", http.StatusInternalServerError)
			return
		}
		subscriptions = append(subscriptions, subscription)
	}

	writeJSON(w, http.StatusOK, subscriptions)
}

func updateWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook subscription ID", http.StatusBadRequest)
		return
	}

	var subscription WebhookSubscription
	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if subscription.WebhookEndpointID == 0 || subscription.EventType == "" {
		http.Error(w, "webhook_endpoint_id and event_type are required", http.StatusBadRequest)
		return
	}

	subscription.ID = id
	subscription.TenantID = claims.TenantID
	err = dbPool.QueryRow(r.Context(), `
		UPDATE webhook_subscriptions ws
		SET webhook_endpoint_id = we.id, event_type = $1, is_active = $2
		FROM webhook_endpoints we
		WHERE ws.id = $3
		  AND ws.tenant_id = $4
		  AND we.id = $5
		  AND we.tenant_id = $4
		RETURNING ws.created_at`,
		subscription.EventType, subscription.IsActive, subscription.ID, subscription.TenantID, subscription.WebhookEndpointID,
	).Scan(&subscription.CreatedAt)
	if err == pgx.ErrNoRows {
		http.Error(w, "Webhook subscription not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating webhook subscription: %v\n", err)
		http.Error(w, "Could not update webhook subscription", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func deleteWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook subscription ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		DELETE FROM webhook_subscriptions
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error deleting webhook subscription: %v\n", err)
		http.Error(w, "Could not delete webhook subscription", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Webhook subscription not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	eventTypeFilter := r.URL.Query().Get("event_type")
	endpointIDFilter := r.URL.Query().Get("webhook_endpoint_id")

	query := `
		SELECT wd.id, wd.tenant_id, wd.webhook_endpoint_id, wd.outbox_event_id, wd.status, wd.attempt_count,
		       wd.http_status, wd.last_error, wd.response_body, wd.delivered_at, wd.created_at
		FROM webhook_deliveries wd
		LEFT JOIN outbox_events oe ON oe.id = wd.outbox_event_id
		WHERE wd.tenant_id = $1`
	args := []interface{}{claims.TenantID}
	argPos := 2

	if statusFilter != "" {
		query += ` AND wd.status = $` + strconv.Itoa(argPos)
		args = append(args, statusFilter)
		argPos++
	}
	if eventTypeFilter != "" {
		query += ` AND oe.event_type = $` + strconv.Itoa(argPos)
		args = append(args, eventTypeFilter)
		argPos++
	}
	if endpointIDFilter != "" {
		endpointID, err := strconv.ParseInt(endpointIDFilter, 10, 64)
		if err != nil {
			http.Error(w, "Invalid webhook_endpoint_id", http.StatusBadRequest)
			return
		}
		query += ` AND wd.webhook_endpoint_id = $` + strconv.Itoa(argPos)
		args = append(args, endpointID)
		argPos++
	}

	query += ` ORDER BY wd.created_at DESC LIMIT 100`

	rows, err := dbPool.Query(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying webhook deliveries: %v\n", err)
		http.Error(w, "Could not fetch webhook deliveries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var deliveries []WebhookDelivery
	for rows.Next() {
		var delivery WebhookDelivery
		if err := rows.Scan(
			&delivery.ID, &delivery.TenantID, &delivery.WebhookEndpointID, &delivery.OutboxEventID,
			&delivery.Status, &delivery.AttemptCount, &delivery.HTTPStatus, &delivery.LastError,
			&delivery.ResponseBody, &delivery.DeliveredAt, &delivery.CreatedAt,
		); err != nil {
			log.Printf("Error scanning webhook delivery row: %v\n", err)
			http.Error(w, "Could not fetch webhook deliveries", http.StatusInternalServerError)
			return
		}
		deliveries = append(deliveries, delivery)
	}

	writeJSON(w, http.StatusOK, deliveries)
}

func getWebhookDeliveryStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var stats WebhookDeliveryStats

	statusRows, err := dbPool.Query(r.Context(), `
		SELECT status, COUNT(*)
		FROM webhook_deliveries
		WHERE tenant_id = $1
		GROUP BY status
		ORDER BY status ASC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying webhook delivery status stats: %v\n", err)
		http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
		return
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var item WebhookDeliveryStatusCount
		if err := statusRows.Scan(&item.Status, &item.Count); err != nil {
			log.Printf("Error scanning webhook delivery status stat: %v\n", err)
			http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
			return
		}
		stats.ByStatus = append(stats.ByStatus, item)
	}

	endpointRows, err := dbPool.Query(r.Context(), `
		SELECT wd.webhook_endpoint_id, COALESCE(we.name, 'unknown'), COUNT(*)
		FROM webhook_deliveries wd
		LEFT JOIN webhook_endpoints we ON we.id = wd.webhook_endpoint_id
		WHERE wd.tenant_id = $1
		GROUP BY wd.webhook_endpoint_id, we.name
		ORDER BY COUNT(*) DESC, wd.webhook_endpoint_id ASC
		LIMIT 20`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying webhook delivery endpoint stats: %v\n", err)
		http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
		return
	}
	defer endpointRows.Close()

	for endpointRows.Next() {
		var item WebhookDeliveryEndpointCount
		if err := endpointRows.Scan(&item.WebhookEndpointID, &item.EndpointName, &item.Count); err != nil {
			log.Printf("Error scanning webhook delivery endpoint stat: %v\n", err)
			http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
			return
		}
		stats.ByEndpoint = append(stats.ByEndpoint, item)
	}

	recentFailed, err := loadWebhookDeliveries(r, claims.TenantID, `
		SELECT id, tenant_id, webhook_endpoint_id, outbox_event_id, status, attempt_count,
		       http_status, last_error, response_body, delivered_at, created_at
		FROM webhook_deliveries
		WHERE tenant_id = $1 AND status = 'failed'
		ORDER BY created_at DESC
		LIMIT 10`)
	if err != nil {
		log.Printf("Error querying recent failed webhook deliveries: %v\n", err)
		http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
		return
	}
	stats.RecentFailed = recentFailed

	recentDelivered, err := loadWebhookDeliveries(r, claims.TenantID, `
		SELECT id, tenant_id, webhook_endpoint_id, outbox_event_id, status, attempt_count,
		       http_status, last_error, response_body, delivered_at, created_at
		FROM webhook_deliveries
		WHERE tenant_id = $1 AND status = 'delivered'
		ORDER BY delivered_at DESC NULLS LAST, created_at DESC
		LIMIT 10`)
	if err != nil {
		log.Printf("Error querying recent delivered webhook deliveries: %v\n", err)
		http.Error(w, "Could not fetch webhook delivery stats", http.StatusInternalServerError)
		return
	}
	stats.RecentDelivered = recentDelivered

	writeJSON(w, http.StatusOK, stats)
}

func getOutboxEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	eventTypeFilter := r.URL.Query().Get("event_type")
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = normalizeReplayLimit(parsedLimit)
	}

	query := `
		SELECT id, tenant_id, event_type, payload, status, attempt_count, next_attempt_at, last_error, created_at, processed_at
		FROM outbox_events
		WHERE tenant_id = $1`
	args := []interface{}{claims.TenantID}
	argPos := 2

	if statusFilter != "" {
		query += ` AND status = $` + strconv.Itoa(argPos)
		args = append(args, statusFilter)
		argPos++
	}
	if eventTypeFilter != "" {
		query += ` AND event_type = $` + strconv.Itoa(argPos)
		args = append(args, eventTypeFilter)
		argPos++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argPos)
	args = append(args, limit)

	rows, err := dbPool.Query(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying outbox events: %v\n", err)
		http.Error(w, "Could not fetch outbox events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID, &event.TenantID, &event.EventType, &event.Payload, &event.Status,
			&event.AttemptCount, &event.NextAttemptAt, &event.LastError, &event.CreatedAt, &event.ProcessedAt,
		); err != nil {
			log.Printf("Error scanning outbox event row: %v\n", err)
			http.Error(w, "Could not fetch outbox events", http.StatusInternalServerError)
			return
		}
		events = append(events, event)
	}

	writeJSON(w, http.StatusOK, events)
}

func loadWebhookDeliveries(r *http.Request, tenantID int64, query string) ([]WebhookDelivery, error) {
	rows, err := dbPool.Query(r.Context(), query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []WebhookDelivery
	for rows.Next() {
		var delivery WebhookDelivery
		if err := rows.Scan(
			&delivery.ID, &delivery.TenantID, &delivery.WebhookEndpointID, &delivery.OutboxEventID,
			&delivery.Status, &delivery.AttemptCount, &delivery.HTTPStatus, &delivery.LastError,
			&delivery.ResponseBody, &delivery.DeliveredAt, &delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}

	return deliveries, rows.Err()
}

func getOutboxEventByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid outbox event ID", http.StatusBadRequest)
		return
	}

	var event OutboxEvent
	err = dbPool.QueryRow(r.Context(), `
		SELECT id, tenant_id, event_type, payload, status, attempt_count, next_attempt_at, last_error, created_at, processed_at
		FROM outbox_events
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID).Scan(
		&event.ID, &event.TenantID, &event.EventType, &event.Payload, &event.Status,
		&event.AttemptCount, &event.NextAttemptAt, &event.LastError, &event.CreatedAt, &event.ProcessedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Outbox event not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying outbox event by ID: %v\n", err)
		http.Error(w, "Could not fetch outbox event", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, event)
}

func getOutboxStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var stats OutboxStats

	statusRows, err := dbPool.Query(r.Context(), `
		SELECT status, COUNT(*)
		FROM outbox_events
		WHERE tenant_id = $1
		GROUP BY status
		ORDER BY status ASC`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying outbox status stats: %v\n", err)
		http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
		return
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var item OutboxStatusCount
		if err := statusRows.Scan(&item.Status, &item.Count); err != nil {
			log.Printf("Error scanning outbox status stat: %v\n", err)
			http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
			return
		}
		stats.ByStatus = append(stats.ByStatus, item)
	}

	typeRows, err := dbPool.Query(r.Context(), `
		SELECT event_type, COUNT(*)
		FROM outbox_events
		WHERE tenant_id = $1
		GROUP BY event_type
		ORDER BY COUNT(*) DESC, event_type ASC
		LIMIT 20`, claims.TenantID)
	if err != nil {
		log.Printf("Error querying outbox event type stats: %v\n", err)
		http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
		return
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var item OutboxEventTypeCount
		if err := typeRows.Scan(&item.EventType, &item.Count); err != nil {
			log.Printf("Error scanning outbox event type stat: %v\n", err)
			http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
			return
		}
		stats.ByEventType = append(stats.ByEventType, item)
	}

	recentFailed, err := loadOutboxEvents(r, claims.TenantID, `
		SELECT id, tenant_id, event_type, payload, status, attempt_count, next_attempt_at, last_error, created_at, processed_at
		FROM outbox_events
		WHERE tenant_id = $1 AND status = 'failed'
		ORDER BY created_at DESC
		LIMIT 10`)
	if err != nil {
		log.Printf("Error querying recent failed outbox events: %v\n", err)
		http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
		return
	}
	stats.RecentFailed = recentFailed

	nextRetry, err := loadOutboxEvents(r, claims.TenantID, `
		SELECT id, tenant_id, event_type, payload, status, attempt_count, next_attempt_at, last_error, created_at, processed_at
		FROM outbox_events
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY next_attempt_at ASC, created_at ASC
		LIMIT 10`)
	if err != nil {
		log.Printf("Error querying next retry outbox events: %v\n", err)
		http.Error(w, "Could not fetch outbox stats", http.StatusInternalServerError)
		return
	}
	stats.NextRetry = nextRetry

	writeJSON(w, http.StatusOK, stats)
}

func loadOutboxEvents(r *http.Request, tenantID int64, query string) ([]OutboxEvent, error) {
	rows, err := dbPool.Query(r.Context(), query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID, &event.TenantID, &event.EventType, &event.Payload, &event.Status,
			&event.AttemptCount, &event.NextAttemptAt, &event.LastError, &event.CreatedAt, &event.ProcessedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func replayWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook delivery ID", http.StatusBadRequest)
		return
	}

	var outboxEventID int64
	err = dbPool.QueryRow(r.Context(), `
		SELECT outbox_event_id
		FROM webhook_deliveries
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID).Scan(&outboxEventID)
	if err == pgx.ErrNoRows {
		http.Error(w, "Webhook delivery not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error loading webhook delivery for replay: %v\n", err)
		http.Error(w, "Could not replay webhook delivery", http.StatusInternalServerError)
		return
	}

	_, err = dbPool.Exec(r.Context(), `
		UPDATE webhook_deliveries
		SET status = 'pending',
		    last_error = NULL,
		    http_status = NULL,
		    response_body = NULL,
		    delivered_at = NULL
		WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		log.Printf("Error resetting webhook delivery state: %v\n", err)
		http.Error(w, "Could not replay webhook delivery", http.StatusInternalServerError)
		return
	}

	_, err = dbPool.Exec(r.Context(), `
		UPDATE outbox_events
		SET status = 'pending',
		    next_attempt_at = NOW(),
		    last_error = NULL,
		    processed_at = NULL
		WHERE id = $1`, outboxEventID)
	if err != nil {
		log.Printf("Error resetting outbox event for replay: %v\n", err)
		http.Error(w, "Could not replay webhook delivery", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"webhook_delivery_id": id,
		"outbox_event_id":     outboxEventID,
		"status":              "requeued",
	})
}

func replayOutboxEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid outbox event ID", http.StatusBadRequest)
		return
	}

	var tenantID int64
	err = dbPool.QueryRow(r.Context(), `
		SELECT COALESCE(tenant_id, 0)
		FROM outbox_events
		WHERE id = $1`, id).Scan(&tenantID)
	if err == pgx.ErrNoRows {
		http.Error(w, "Outbox event not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error loading outbox event for replay: %v\n", err)
		http.Error(w, "Could not replay outbox event", http.StatusInternalServerError)
		return
	}

	if tenantID != 0 && tenantID != claims.TenantID {
		http.Error(w, "Outbox event not found", http.StatusNotFound)
		return
	}

	_, err = dbPool.Exec(r.Context(), `
		UPDATE outbox_events
		SET status = 'pending',
		    next_attempt_at = NOW(),
		    last_error = NULL,
		    processed_at = NULL
		WHERE id = $1`, id)
	if err != nil {
		log.Printf("Error requeueing outbox event: %v\n", err)
		http.Error(w, "Could not replay outbox event", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"outbox_event_id": id,
		"status":          "requeued",
	})
}

func replayOutboxEventsByFilter(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		EventType string `json:"event_type"`
		Status    string `json:"status"`
		Limit     int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if input.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}
	input.Limit = normalizeReplayLimit(input.Limit)
	if input.Status == "" {
		input.Status = "processed"
	}

	commandTag, err := dbPool.Exec(r.Context(), `
		UPDATE outbox_events
		SET status = 'pending',
		    next_attempt_at = NOW(),
		    last_error = NULL,
		    processed_at = NULL
		WHERE id IN (
			SELECT id
			FROM outbox_events
			WHERE tenant_id = $1
			  AND event_type = $2
			  AND status = $3
			ORDER BY created_at DESC
			LIMIT $4
		)`,
		claims.TenantID, input.EventType, input.Status, input.Limit,
	)
	if err != nil {
		log.Printf("Error replaying outbox events by filter: %v\n", err)
		http.Error(w, "Could not replay outbox events", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_type":     input.EventType,
		"previous_status": input.Status,
		"requeued_count": commandTag.RowsAffected(),
	})
}

func getAuditLog(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	entityTypeFilter := r.URL.Query().Get("entity_type")
	actionFilter := r.URL.Query().Get("action")
	limit := normalizeReplayLimit(0)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = normalizeReplayLimit(parsedLimit)
	}

	query := `
		SELECT id, tenant_id, actor_type, actor_id, action, entity_type, entity_id,
		       changes_json, ip_address::text, user_agent, created_at
		FROM audit_log
		WHERE tenant_id = $1`
	args := []interface{}{claims.TenantID}
	argPos := 2

	if entityTypeFilter != "" {
		query += ` AND entity_type = $` + strconv.Itoa(argPos)
		args = append(args, entityTypeFilter)
		argPos++
	}
	if actionFilter != "" {
		query += ` AND action = $` + strconv.Itoa(argPos)
		args = append(args, actionFilter)
		argPos++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argPos)
	args = append(args, limit)

	rows, err := dbPool.Query(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying audit log: %v\n", err)
		http.Error(w, "Could not fetch audit log", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.ActorType, &entry.ActorID, &entry.Action, &entry.EntityType,
			&entry.EntityID, &entry.Changes, &entry.IPAddress, &entry.UserAgent, &entry.CreatedAt,
		); err != nil {
			log.Printf("Error scanning audit log row: %v\n", err)
			http.Error(w, "Could not fetch audit log", http.StatusInternalServerError)
			return
		}
		entries = append(entries, entry)
	}

	writeJSON(w, http.StatusOK, entries)
}

func normalizeReplayLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 25
	}
	return limit
}
