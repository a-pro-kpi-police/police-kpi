package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type adminUserFields struct {
	Username             string
	Password             string
	FullName             string
	Rank                 string
	Role                 string
	DistrictID           *int
	SourceEmployeeNumber string
	MonthlyNorm          float64
}

type sqlScanner interface {
	Scan(dest ...any) error
}

const adminUserSelect = `SELECT
	u.id, u.username, u.full_name, u.rank, u.role,
	u.district_id, u.source_employee_number, u.monthly_norm,
	d.id, d.name, d.coeff,
	COALESCE(e.entries_count, 0)
FROM users u
LEFT JOIN districts d ON d.id = u.district_id
LEFT JOIN (
	SELECT officer_id, COUNT(*) AS entries_count
	FROM kpi_entries
	GROUP BY officer_id
) e ON e.officer_id = u.id`

func (a *App) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	requestedDistrictID, err := parseOptionalInt(r.URL.Query().Get("district_id"))
	if err != nil || (requestedDistrictID != nil && *requestedDistrictID <= 0) {
		writeError(w, http.StatusBadRequest, "Участок указан неверно")
		return
	}
	users, err := a.listAdminUsers(r.Context(), requestedDistrictID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *App) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	var payload AdminUserSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные сотрудника")
		return
	}
	fields, err := a.prepareAdminUserFields(r.Context(), payload, nil, true)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := a.ensureAdminUserUnique(r.Context(), fields, 0); err != nil {
		writeHTTPError(w, err)
		return
	}
	passwordHash, err := hashPassword(fields.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var createdID int
	err = a.db.QueryRowContext(r.Context(), `INSERT INTO users
		(username, password_hash, full_name, rank, role, district_id, source_employee_number, monthly_norm)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		fields.Username,
		passwordHash,
		fields.FullName,
		fields.Rank,
		fields.Role,
		nullableInt(fields.DistrictID),
		nullableString(fields.SourceEmployeeNumber),
		fields.MonthlyNorm,
	).Scan(&createdID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, err := a.adminUserByID(r.Context(), createdID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (a *App) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	userID, valid := parseIDPathValue(r, "user_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Сотрудник не найден")
		return
	}
	existing, err := a.adminUserByID(r.Context(), userID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	var payload AdminUserSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные сотрудника")
		return
	}
	fields, err := a.prepareAdminUserFields(r.Context(), payload, &existing, false)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := a.ensureAdminUserUnique(r.Context(), fields, userID); err != nil {
		writeHTTPError(w, err)
		return
	}

	if strings.TrimSpace(fields.Password) != "" {
		passwordHash, err := hashPassword(fields.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, err = a.db.ExecContext(r.Context(), `UPDATE users
			SET username = ?, password_hash = ?, full_name = ?, rank = ?, role = ?, district_id = ?,
				source_employee_number = ?, monthly_norm = ?
			WHERE id = ? AND role IN (?, ?)`,
			fields.Username,
			passwordHash,
			fields.FullName,
			fields.Rank,
			fields.Role,
			nullableInt(fields.DistrictID),
			nullableString(fields.SourceEmployeeNumber),
			fields.MonthlyNorm,
			userID,
			roleHead,
			roleOfficer,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		_, err = a.db.ExecContext(r.Context(), `UPDATE users
			SET username = ?, full_name = ?, rank = ?, role = ?, district_id = ?,
				source_employee_number = ?, monthly_norm = ?
			WHERE id = ? AND role IN (?, ?)`,
			fields.Username,
			fields.FullName,
			fields.Rank,
			fields.Role,
			nullableInt(fields.DistrictID),
			nullableString(fields.SourceEmployeeNumber),
			fields.MonthlyNorm,
			userID,
			roleHead,
			roleOfficer,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	updated, err := a.adminUserByID(r.Context(), userID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	userID, valid := parseIDPathValue(r, "user_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Сотрудник не найден")
		return
	}
	if _, err := a.adminUserByID(r.Context(), userID); err != nil {
		writeHTTPError(w, err)
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	attachments, err := attachmentNamesForOfficer(r.Context(), tx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM direct_messages
		WHERE sender_id = ? OR recipient_id = ? OR entry_id IN (
			SELECT id FROM kpi_entries WHERE officer_id = ?
		)`, userID, userID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM support_messages
		WHERE sender_id = ? OR conversation_id IN (
			SELECT id FROM support_conversations WHERE user_id = ?
		)`, userID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM support_conversations WHERE user_id = ?`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM app_notifications WHERE user_id = ?`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM kpi_targets WHERE officer_id = ?`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM kpi_entries WHERE officer_id = ?`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := tx.ExecContext(r.Context(), `DELETE FROM users WHERE id = ? AND role IN (?, ?)`, userID, roleHead, roleOfficer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		writeError(w, http.StatusNotFound, "Сотрудник не найден")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.removeStoredAttachments(attachments)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) listAdminUsers(ctx context.Context, districtID *int) ([]AdminUserOut, error) {
	sqlText := adminUserSelect + ` WHERE u.role IN (?, ?)`
	args := []any{roleHead, roleOfficer}
	if districtID != nil {
		sqlText += ` AND u.district_id = ?`
		args = append(args, *districtID)
	}
	sqlText += ` ORDER BY u.full_name`

	rows, err := a.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []AdminUserOut{}
	for rows.Next() {
		user, err := scanAdminUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (a *App) adminUserByID(ctx context.Context, id int) (AdminUserOut, error) {
	user, err := scanAdminUser(a.db.QueryRowContext(ctx, adminUserSelect+` WHERE u.id = ? AND u.role IN (?, ?)`, id, roleHead, roleOfficer))
	if errors.Is(err, sql.ErrNoRows) {
		return AdminUserOut{}, httpErr{status: http.StatusNotFound, detail: "Сотрудник не найден"}
	}
	return user, err
}

func scanAdminUser(scanner sqlScanner) (AdminUserOut, error) {
	var user AdminUserOut
	var districtID sql.NullInt64
	var sourceEmployeeNumber sql.NullString
	var joinedDistrictID sql.NullInt64
	var districtName sql.NullString
	var districtCoeff sql.NullFloat64
	if err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.FullName,
		&user.Rank,
		&user.Role,
		&districtID,
		&sourceEmployeeNumber,
		&user.MonthlyNorm,
		&joinedDistrictID,
		&districtName,
		&districtCoeff,
		&user.EntriesCount,
	); err != nil {
		return AdminUserOut{}, err
	}
	user.MonthlyNorm = round2(user.MonthlyNorm)
	if sourceEmployeeNumber.Valid {
		user.SourceEmployeeNumber = sourceEmployeeNumber.String
	}
	if joinedDistrictID.Valid {
		user.District = &District{
			ID:    int(joinedDistrictID.Int64),
			Name:  districtName.String,
			Coeff: districtCoeff.Float64,
		}
	}
	return user, nil
}

func (a *App) prepareAdminUserFields(ctx context.Context, payload AdminUserSavePayload, existing *AdminUserOut, requirePassword bool) (adminUserFields, error) {
	username := strings.TrimSpace(payload.Username)
	if username == "" {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "Логин обязателен"}
	}
	password := strings.TrimSpace(payload.Password)
	if requirePassword && password == "" {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "Пароль обязателен"}
	}
	fullName := strings.TrimSpace(payload.FullName)
	if fullName == "" {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "ФИО сотрудника обязательно"}
	}
	role := strings.TrimSpace(payload.Role)
	if role == "" && existing != nil {
		role = existing.Role
	}
	if role == "" {
		role = roleOfficer
	}
	if role != roleHead && role != roleOfficer {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "Недопустимая роль пользователя"}
	}
	districtID := payload.DistrictID
	if districtID != nil && *districtID <= 0 {
		districtID = nil
	}
	if role == roleOfficer && districtID == nil {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "Укажите участок сотрудника"}
	}
	if districtID != nil {
		if err := a.ensureDistrictExists(ctx, *districtID); err != nil {
			return adminUserFields{}, err
		}
	}

	monthlyNorm := 100.0
	if existing != nil {
		monthlyNorm = existing.MonthlyNorm
	}
	if payload.MonthlyNorm != nil {
		monthlyNorm = *payload.MonthlyNorm
	}
	if monthlyNorm <= 0 {
		return adminUserFields{}, httpErr{status: http.StatusBadRequest, detail: "Месячная норма должна быть больше нуля"}
	}

	return adminUserFields{
		Username:             username,
		Password:             password,
		FullName:             fullName,
		Rank:                 strings.TrimSpace(payload.Rank),
		Role:                 role,
		DistrictID:           districtID,
		SourceEmployeeNumber: strings.TrimSpace(payload.SourceEmployeeNumber),
		MonthlyNorm:          monthlyNorm,
	}, nil
}

func (a *App) ensureDistrictExists(ctx context.Context, districtID int) error {
	var found int
	err := a.db.QueryRowContext(ctx, `SELECT id FROM districts WHERE id = ?`, districtID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return httpErr{status: http.StatusNotFound, detail: "Участок не найден"}
	}
	return err
}

func (a *App) ensureAdminUserUnique(ctx context.Context, fields adminUserFields, excludeID int) error {
	taken, err := valueTaken(ctx, a.db, `SELECT id FROM users WHERE username = ? AND id <> ? LIMIT 1`, fields.Username, excludeID)
	if err != nil {
		return err
	}
	if taken {
		return httpErr{status: http.StatusConflict, detail: "Логин уже занят"}
	}
	if fields.SourceEmployeeNumber == "" {
		return nil
	}
	taken, err = valueTaken(ctx, a.db, `SELECT id FROM users WHERE source_employee_number = ? AND id <> ? LIMIT 1`, fields.SourceEmployeeNumber, excludeID)
	if err != nil {
		return err
	}
	if taken {
		return httpErr{status: http.StatusConflict, detail: "Табельный номер уже занят"}
	}
	return nil
}

func valueTaken(ctx context.Context, q dbtx, query string, value any, excludeID int) (bool, error) {
	var id int
	err := q.QueryRowContext(ctx, query, value, excludeID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func attachmentNamesForOfficer(ctx context.Context, q dbtx, officerID int) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT attachment_storage_name
		FROM kpi_entries
		WHERE officer_id = ? AND attachment_storage_name <> ''
		UNION
		SELECT a.storage_name
		FROM entry_attachments a
		JOIN kpi_entries e ON e.id = a.entry_id
		WHERE e.officer_id = ?`, officerID, officerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (a *App) removeStoredAttachments(names []string) {
	for _, name := range names {
		if !validAttachmentStorageName(name) {
			continue
		}
		_ = os.Remove(filepath.Join(a.uploadDir, name))
	}
}

func validAttachmentStorageName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, `/\`)
}
