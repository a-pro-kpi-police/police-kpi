package bootstrap

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	targetOfficers          = 20
	targetKPIEntries        = 1000
	MonthlyImportMode       = "monthly_import"
	CatalogOnlyMode         = "catalog_only"
	defaultMonthlyImportRel = "import_data/monthly_kpi_2026_07.csv"
	roleAdmin               = "admin"
	roleHead                = "head"
	roleOfficer             = "officer"
	statusPending           = "pending"
	statusConfirmed         = "confirmed"
	statusRejected          = "rejected"
	headFullName            = "Кожахметов Ерсін Берікұлы"
)

type Tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	Commit() error
	Rollback() error
}

type BeginTxFunc func(context.Context, *sql.TxOptions) (Tx, error)

type HashPasswordFunc func(string) (string, error)

type RecalculateFunc func(context.Context, Tx, int, string) error

type Config struct {
	BeginTx                 BeginTxFunc
	HashPassword            HashPasswordFunc
	RecalculateOfficerMonth RecalculateFunc
}

type District struct {
	ID    int
	Name  string
	Coeff float64
}

type User struct {
	ID int
}

type CatalogItem struct {
	ID            int
	Code          string
	Indicator     string
	Block         string
	Subblock      string
	NPA           string
	PointsPerUnit float64
	Unit          string
	SourceURL     string
	MonthlyLimit  float64
	IsMass        bool
	IsCritical    bool
	IsActive      bool
}

var demoDistricts = []District{
	{Name: "Обычный участок", Coeff: 1.0},
	{Name: "Повышенной сложности", Coeff: 1.1},
	{Name: "Сложный участок", Coeff: 1.2},
	{Name: "Особо сложный участок", Coeff: 1.3},
}

func Seed(ctx context.Context, config Config) error {
	if config.BeginTx == nil {
		return errors.New("bootstrap begin tx is required")
	}
	if config.HashPassword == nil {
		return errors.New("bootstrap hash password func is required")
	}
	if config.RecalculateOfficerMonth == nil {
		return errors.New("bootstrap scoring recalculation func is required")
	}

	tx, err := config.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	catalog, err := loadCatalogCSV("kpi_catalog.csv")
	if err != nil {
		return err
	}
	if err := syncCatalog(ctx, tx, catalog); err != nil {
		return err
	}
	if err := ensureAdmin(ctx, tx, config); err != nil {
		return err
	}

	if os.Getenv("KPI_SEED_MODE") == CatalogOnlyMode {
		return tx.Commit()
	}

	if os.Getenv("KPI_SEED_MODE") == MonthlyImportMode {
		if err := seedMonthlyImport(ctx, tx, catalog, config); err != nil {
			return err
		}
		if err := ensureTargets(ctx, tx); err != nil {
			return err
		}
		return tx.Commit()
	}

	var usersCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role <> ?`, roleAdmin).Scan(&usersCount); err != nil {
		return err
	}
	if usersCount == 0 {
		if err := seedDemoData(ctx, tx, config); err != nil {
			return err
		}
	}
	if err := ensureTargets(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func loadCatalogCSV(path string) ([]CatalogItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	index := map[string]int{}
	for i, header := range headers {
		index[header] = i
	}
	expected := []string{
		"code", "block", "subblock", "indicator", "npa", "points_per_unit",
		"source_url", "unit", "monthly_limit", "is_mass", "is_critical",
	}
	for _, name := range expected {
		if _, ok := index[name]; !ok {
			return nil, fmt.Errorf("invalid kpi_catalog.csv: missing %s", name)
		}
	}

	var items []CatalogItem
	seen := map[string]bool{}
	for rowNumber := 2; ; rowNumber++ {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		code := strings.TrimSpace(record[index["code"]])
		if code == "" {
			return nil, fmt.Errorf("empty KPI code in kpi_catalog.csv row %d", rowNumber)
		}
		if seen[code] {
			return nil, fmt.Errorf("duplicate KPI code %s", code)
		}
		seen[code] = true
		points, err := strconv.ParseFloat(strings.TrimSpace(record[index["points_per_unit"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid points_per_unit in row %d: %w", rowNumber, err)
		}
		limit, err := parseFloat(record[index["monthly_limit"]], 0)
		if err != nil {
			return nil, fmt.Errorf("invalid monthly_limit in row %d: %w", rowNumber, err)
		}
		unit := strings.TrimSpace(record[index["unit"]])
		if unit == "" {
			return nil, fmt.Errorf("empty unit in kpi_catalog.csv row %d", rowNumber)
		}
		items = append(items, CatalogItem{
			Code:          code,
			Block:         strings.TrimSpace(record[index["block"]]),
			Subblock:      strings.TrimSpace(record[index["subblock"]]),
			Indicator:     strings.TrimSpace(record[index["indicator"]]),
			NPA:           strings.TrimSpace(record[index["npa"]]),
			PointsPerUnit: points,
			SourceURL:     strings.TrimSpace(record[index["source_url"]]),
			Unit:          unit,
			MonthlyLimit:  limit,
			IsMass:        parseBoolInt(record[index["is_mass"]]),
			IsCritical:    parseBoolInt(record[index["is_critical"]]),
			IsActive:      true,
		})
	}
	return items, nil
}

func syncCatalog(ctx context.Context, tx Tx, catalog []CatalogItem) error {
	if _, err := tx.ExecContext(ctx, `UPDATE kpi_catalog SET is_active = 0`); err != nil {
		return err
	}
	for _, item := range catalog {
		if _, err := tx.ExecContext(ctx, `INSERT INTO kpi_catalog
			(code, indicator, block, subblock, npa, points_per_unit, unit, source_url, monthly_limit, is_mass, is_critical, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
			ON CONFLICT(code) DO UPDATE SET
				indicator = excluded.indicator,
				block = excluded.block,
				subblock = excluded.subblock,
				npa = excluded.npa,
				points_per_unit = excluded.points_per_unit,
				unit = excluded.unit,
				source_url = excluded.source_url,
				monthly_limit = excluded.monthly_limit,
				is_mass = excluded.is_mass,
				is_critical = excluded.is_critical,
				is_active = 1`,
			item.Code,
			item.Indicator,
			item.Block,
			item.Subblock,
			item.NPA,
			item.PointsPerUnit,
			item.Unit,
			item.SourceURL,
			item.MonthlyLimit,
			boolInt(item.IsMass),
			boolInt(item.IsCritical),
		); err != nil {
			return err
		}
	}
	return nil
}

func monthlyImportPath() string {
	if value := os.Getenv("KPI_MONTHLY_IMPORT_PATH"); value != "" {
		return value
	}
	return filepath.FromSlash(defaultMonthlyImportRel)
}

func seedMonthlyImport(ctx context.Context, tx Tx, catalog []CatalogItem, config Config) error {
	path := monthlyImportPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("monthly import file not found: %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM import_batches WHERE source_checksum = ?`, checksum).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	records, headers, err := readCSVFile(path)
	if err != nil {
		return err
	}
	identityColumns := []string{"source_row", "month", "employee_number", "full_name", "position", "coefficient"}
	if len(headers) <= len(identityColumns) {
		return errors.New("invalid monthly import structure")
	}
	for i, name := range identityColumns {
		if headers[i] != name {
			return errors.New("invalid monthly import structure")
		}
	}
	codes := headers[len(identityColumns):]
	if len(codes) != len(catalog) {
		return errors.New("monthly import KPI codes do not match catalog")
	}
	for i, item := range catalog {
		if codes[i] != item.Code {
			return errors.New("monthly import KPI codes do not match catalog")
		}
	}
	if len(records) == 0 {
		return errors.New("monthly import file is empty")
	}
	month := strings.TrimSpace(records[0][1])
	if err := validateMonth(month); err != nil {
		return err
	}
	nameSet := map[string]bool{}
	for rowIndex, record := range records {
		if strings.TrimSpace(record[1]) != month {
			return errors.New("monthly import must contain one month")
		}
		name := strings.TrimSpace(record[3])
		if name == "" || nameSet[name] {
			return errors.New("employee names are empty or duplicated")
		}
		nameSet[name] = true
		for codeIndex := range codes {
			quantity, err := parseFloat(record[len(identityColumns)+codeIndex], 0)
			if err != nil || quantity < 0 {
				return fmt.Errorf("negative or invalid quantity in row %d, %s", rowIndex+2, codes[codeIndex])
			}
		}
	}

	districtID, err := upsertDistrict(ctx, tx, "Батальон патрульной полиции", 1.0)
	if err != nil {
		return err
	}
	if err := upsertHead(ctx, tx, districtID, config); err != nil {
		return err
	}

	catalogByCode, err := catalogByCode(ctx, tx)
	if err != nil {
		return err
	}
	defaultPasswordHash, err := config.HashPassword(envDefault("KPI_IMPORT_DEFAULT_PASSWORD", "pass123"))
	if err != nil {
		return err
	}
	var importedUserIDs []int
	for _, record := range records {
		employeeNumber, err := employeeNumberString(record[2])
		if err != nil {
			return err
		}
		userID, err := upsertImportedUser(ctx, tx, employeeNumber, defaultPasswordHash, strings.TrimSpace(record[3]), strings.TrimSpace(record[4]), districtID)
		if err != nil {
			return err
		}
		importedUserIDs = append(importedUserIDs, userID)
	}

	var batchID int
	err = tx.QueryRowContext(ctx, `INSERT INTO import_batches
		(source_name, source_sheet, source_checksum, month, employee_count, kpi_count, entry_count, imported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		filepath.Base(path),
		"Месячная_таблица-1",
		checksum,
		month,
		len(records),
		len(codes),
		len(records)*len(codes),
		nowDB(),
	).Scan(&batchID)
	if err != nil {
		return err
	}

	sourcePrefix := "numbers:" + month + ":"
	for rowIndex, record := range records {
		employeeNumber, err := employeeNumberString(record[2])
		if err != nil {
			return err
		}
		employeeNumberInt, _ := strconv.Atoi(employeeNumber)
		coefficient, err := parseFloat(record[5], 1)
		if err != nil || coefficient == 0 {
			coefficient = 1
		}
		sourceRow, err := parseIntFromFloat(record[0])
		if err != nil {
			return err
		}
		for codeIndex, code := range codes {
			item := catalogByCode[code]
			quantity, err := parseFloat(record[len(identityColumns)+codeIndex], 0)
			if err != nil {
				return err
			}
			sourceKey := fmt.Sprintf("%s%03d:%s", sourcePrefix, employeeNumberInt, code)
			day := 1 + ((employeeNumberInt*17 + codeIndex*7) % 29)
			entryDate := fmt.Sprintf("%s-%02d", month, day)
			raw := round2(quantity * item.PointsPerUnit)
			comments := fmt.Sprintf("Импорт из Месячная_таблица-1, строка %d; агрегат за %s", sourceRow, month)
			if _, err := tx.ExecContext(ctx, `INSERT INTO kpi_entries
				(date, month, officer_id, catalog_id, quantity, counted_quantity, points_per_unit, coeff,
				 raw_score, capped_score, anti_coeff, final_score, status, comments, head_comment,
				 source_key, source_row, is_monthly_aggregate, import_batch_id)
				VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?, 0, 1.0, 0, ?, ?, ?, ?, ?, 1, ?)
				ON CONFLICT(source_key) DO UPDATE SET
					date = excluded.date,
					month = excluded.month,
					officer_id = excluded.officer_id,
					catalog_id = excluded.catalog_id,
					quantity = excluded.quantity,
					points_per_unit = excluded.points_per_unit,
					coeff = excluded.coeff,
					raw_score = excluded.raw_score,
					status = excluded.status,
					comments = excluded.comments,
					head_comment = excluded.head_comment,
					source_row = excluded.source_row,
					is_monthly_aggregate = excluded.is_monthly_aggregate,
					import_batch_id = excluded.import_batch_id`,
				entryDate,
				month,
				importedUserIDs[rowIndex],
				item.ID,
				quantity,
				item.PointsPerUnit,
				coefficient,
				raw,
				statusConfirmed,
				comments,
				"Подтверждено при импорте месячной таблицы",
				sourceKey,
				sourceRow,
				batchID,
			); err != nil {
				return err
			}
		}
	}

	for _, userID := range importedUserIDs {
		if err := config.RecalculateOfficerMonth(ctx, tx, userID, month); err != nil {
			return err
		}
	}
	return nil
}

func readCSVFile(path string) ([][]string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	return records, headers, nil
}

func upsertDistrict(ctx context.Context, tx Tx, name string, coeff float64) (int, error) {
	if _, err := tx.ExecContext(ctx, `INSERT INTO districts(name, coeff) VALUES(?, ?)
		ON CONFLICT(name) DO UPDATE SET coeff = excluded.coeff`, name, coeff); err != nil {
		return 0, err
	}
	var id int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM districts WHERE name = ?`, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertHead(ctx context.Context, tx Tx, districtID int, config Config) error {
	password, err := config.HashPassword(envDefault("KPI_HEAD_PASSWORD", "head123"))
	if err != nil {
		return err
	}
	var existing int
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE username = 'head'`).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `INSERT INTO users
			(username, password_hash, full_name, rank, role, district_id, monthly_norm)
			VALUES ('head', ?, ?, ?, ?, ?, 100.0)`,
			password,
			headFullName,
			"Начальник подразделения",
			roleHead,
			districtID,
		)
		return err
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE users SET full_name = ?, rank = ?, role = ?, district_id = ? WHERE id = ?`,
		headFullName, "Начальник подразделения", roleHead, districtID, existing)
	return err
}

func ensureAdmin(ctx context.Context, tx Tx, config Config) error {
	password, err := config.HashPassword(envDefault("KPI_ADMIN_PASSWORD", "admin123"))
	if err != nil {
		return err
	}
	var existing int
	var existingRole string
	err = tx.QueryRowContext(ctx, `SELECT id, role FROM users WHERE username = 'admin'`).Scan(&existing, &existingRole)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `INSERT INTO users
			(username, password_hash, full_name, rank, role, district_id, monthly_norm)
			VALUES ('admin', ?, ?, ?, ?, NULL, 100.0)`,
			password,
			"Системный администратор",
			"Администратор",
			roleAdmin,
		)
		return err
	}
	if err != nil {
		return err
	}
	if existingRole != roleAdmin {
		return errors.New("username admin is reserved for the system administrator")
	}
	_, err = tx.ExecContext(ctx, `UPDATE users
		SET full_name = ?, rank = ?, role = ?, district_id = NULL
		WHERE id = ?`,
		"Системный администратор", "Администратор", roleAdmin, existing)
	return err
}

func upsertImportedUser(ctx context.Context, tx Tx, employeeNumber, passwordHash, fullName, rank string, districtID int) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE source_employee_number = ?`, employeeNumber).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		employeeInt, _ := strconv.Atoi(employeeNumber)
		err := tx.QueryRowContext(ctx, `INSERT INTO users
			(username, password_hash, full_name, rank, role, district_id, source_employee_number, monthly_norm)
			VALUES (?, ?, ?, ?, ?, ?, ?, 100.0)
			RETURNING id`,
			fmt.Sprintf("officer%03d", employeeInt),
			passwordHash,
			fullName,
			rank,
			roleOfficer,
			districtID,
			employeeNumber,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE users
		SET full_name = ?, rank = ?, role = ?, district_id = ?, monthly_norm = 100.0
		WHERE id = ?`, fullName, rank, roleOfficer, districtID, id)
	return id, err
}

func employeeNumberString(value string) (string, error) {
	number, err := parseIntFromFloat(value)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(number), nil
}

func parseIntFromFloat(value string) (int, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}

func seedDemoData(ctx context.Context, tx Tx, config Config) error {
	for _, district := range demoDistricts {
		if _, err := upsertDistrict(ctx, tx, district.Name, district.Coeff); err != nil {
			return err
		}
	}
	var districtIDs []int
	rows, err := tx.QueryContext(ctx, `SELECT id FROM districts ORDER BY id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		districtIDs = append(districtIDs, id)
	}
	rows.Close()
	if len(districtIDs) == 0 {
		return errors.New("no districts for demo seed")
	}

	headPassword, err := config.HashPassword("head123")
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users
		(username, password_hash, full_name, rank, role, district_id, monthly_norm)
		VALUES (?, ?, ?, ?, ?, ?, 100.0)`,
		"head", headPassword, headFullName, "Полковник полиции", roleHead, districtIDs[0],
	); err != nil {
		return err
	}

	userIDs := make([]int, 0, targetOfficers)
	for i := 1; i <= targetOfficers; i++ {
		password, err := config.HashPassword("pass123")
		if err != nil {
			return err
		}
		var id int
		err = tx.QueryRowContext(ctx, `INSERT INTO users
			(username, password_hash, full_name, rank, role, district_id, monthly_norm)
			VALUES (?, ?, ?, ?, ?, ?, 100.0)
			RETURNING id`,
			fmt.Sprintf("officer%03d", i),
			password,
			fmt.Sprintf("Сотрудник %03d", i),
			demoRank(i),
			roleOfficer,
			districtIDs[(i-1)%len(districtIDs)],
		).Scan(&id)
		if err != nil {
			return err
		}
		userIDs = append(userIDs, id)
	}

	catalog, err := queryActivePositiveCatalog(ctx, tx)
	if err != nil {
		return err
	}
	if len(catalog) == 0 {
		return errors.New("no active positive KPI catalog for demo seed")
	}
	statuses := []string{statusConfirmed, statusConfirmed, statusConfirmed, statusConfirmed, statusConfirmed, statusConfirmed, statusPending, statusPending, statusPending, statusRejected}
	today := time.Now()
	affectedMonths := map[string]map[int]bool{}
	for i := 0; i < targetKPIEntries; i++ {
		userID := userIDs[i%len(userIDs)]
		item := catalog[(i*7)%len(catalog)]
		daysAgo := (i * 11) % 365
		date := today.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		month := monthFromDate(date)
		status := statuses[i%len(statuses)]
		quantity := float64(1 + (i % 3))
		coeff := demoDistricts[i%len(demoDistricts)].Coeff
		counted := quantity
		if item.MonthlyLimit > 0 && counted > item.MonthlyLimit {
			counted = item.MonthlyLimit
		}
		raw := round2(quantity * item.PointsPerUnit)
		capped := round2(counted * item.PointsPerUnit)
		final := round2(capped * coeff)
		if _, err := tx.ExecContext(ctx, `INSERT INTO kpi_entries
			(date, month, officer_id, catalog_id, quantity, counted_quantity, points_per_unit, coeff,
			 raw_score, capped_score, anti_coeff, final_score, status, comments, head_comment)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1.0, ?, ?, '', ?)`,
			date,
			month,
			userID,
			item.ID,
			quantity,
			counted,
			item.PointsPerUnit,
			coeff,
			raw,
			capped,
			final,
			status,
			demoHeadComment(status),
		); err != nil {
			return err
		}
		if affectedMonths[month] == nil {
			affectedMonths[month] = map[int]bool{}
		}
		affectedMonths[month][userID] = true
	}
	for month, officers := range affectedMonths {
		for officerID := range officers {
			if err := config.RecalculateOfficerMonth(ctx, tx, officerID, month); err != nil {
				return err
			}
		}
	}
	return nil
}

func demoRank(index int) string {
	ranks := []string{
		"Лейтенант полиции",
		"Старший лейтенант полиции",
		"Капитан полиции",
		"Майор полиции",
	}
	return ranks[index%len(ranks)]
}

func demoHeadComment(status string) string {
	switch status {
	case statusConfirmed:
		return "Подтверждено"
	case statusRejected:
		return "Недостаточно данных"
	default:
		return ""
	}
}

func ensureTargets(ctx context.Context, tx Tx) error {
	officers, err := listOfficerUsers(ctx, tx, 0)
	if err != nil {
		return err
	}
	catalog, err := queryActivePositiveCatalog(ctx, tx)
	if err != nil {
		return err
	}
	if len(officers) == 0 || len(catalog) == 0 {
		return nil
	}
	months := []string{monthShift(time.Now(), 0), monthShift(time.Now(), -1)}
	existing := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT month, officer_id, catalog_id FROM kpi_targets WHERE month IN (?, ?)`, months[0], months[1])
	if err != nil {
		return err
	}
	for rows.Next() {
		var month string
		var officerID, catalogID int
		if err := rows.Scan(&month, &officerID, &catalogID); err != nil {
			rows.Close()
			return err
		}
		existing[targetKey(month, officerID, catalogID)] = true
	}
	rows.Close()

	confirmed := map[string]float64{}
	rows, err = tx.QueryContext(ctx, `SELECT month, officer_id, catalog_id, SUM(quantity)
		FROM kpi_entries
		WHERE status = ? AND month IN (?, ?)
		GROUP BY month, officer_id, catalog_id`, statusConfirmed, months[0], months[1])
	if err != nil {
		return err
	}
	for rows.Next() {
		var month string
		var officerID, catalogID int
		var quantity sql.NullFloat64
		if err := rows.Scan(&month, &officerID, &catalogID, &quantity); err != nil {
			rows.Close()
			return err
		}
		confirmed[targetKey(month, officerID, catalogID)] = round2(quantity.Float64)
	}
	rows.Close()

	for officerIndex, officer := range officers {
		for monthIndex, month := range months {
			for catalogIndex, item := range catalog {
				key := targetKey(month, officer.ID, item.ID)
				if existing[key] {
					continue
				}
				quantity := defaultTargetQuantity(confirmed[key], officerIndex, catalogIndex, monthIndex)
				if _, err := tx.ExecContext(ctx, `INSERT INTO kpi_targets(month, officer_id, catalog_id, target_quantity)
					VALUES (?, ?, ?, ?)`, month, officer.ID, item.ID, quantity); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func targetKey(month string, officerID, catalogID int) string {
	return fmt.Sprintf("%s:%d:%d", month, officerID, catalogID)
}

func defaultTargetQuantity(confirmedQuantity float64, officerIndex, catalogIndex, monthIndex int) float64 {
	profile := officerIndex % 3
	if confirmedQuantity > 0 {
		switch profile {
		case 0:
			return round2(confirmedQuantity)
		case 1:
			return round2(maxFloat(confirmedQuantity-0.5, 0.5))
		default:
			extra := 0.5
			if (catalogIndex+monthIndex)%2 == 0 {
				extra = 1.0
			}
			return round2(confirmedQuantity + extra)
		}
	}
	if profile == 2 && (catalogIndex+monthIndex)%4 == 0 {
		return 1.0
	}
	return 0.0
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func queryActivePositiveCatalog(ctx context.Context, q queryer) ([]CatalogItem, error) {
	return queryCatalog(ctx, q, `SELECT
		id, code, indicator, block, subblock, npa, points_per_unit, unit,
		source_url, monthly_limit, is_mass, is_critical, is_active
	FROM kpi_catalog
	WHERE is_active = 1 AND points_per_unit >= 0
	ORDER BY block, code`)
}

func catalogByCode(ctx context.Context, q queryer) (map[string]CatalogItem, error) {
	items, err := queryCatalog(ctx, q, `SELECT
		id, code, indicator, block, subblock, npa, points_per_unit, unit,
		source_url, monthly_limit, is_mass, is_critical, is_active
	FROM kpi_catalog
	ORDER BY block, code`)
	if err != nil {
		return nil, err
	}
	result := map[string]CatalogItem{}
	for _, item := range items {
		result[item.Code] = item
	}
	return result, nil
}

func queryCatalog(ctx context.Context, q queryer, query string, args ...any) ([]CatalogItem, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CatalogItem
	for rows.Next() {
		item, err := scanCatalogItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanCatalogItem(scanner interface {
	Scan(dest ...any) error
}) (CatalogItem, error) {
	var item CatalogItem
	var isMass, isCritical, isActive int
	err := scanner.Scan(
		&item.ID,
		&item.Code,
		&item.Indicator,
		&item.Block,
		&item.Subblock,
		&item.NPA,
		&item.PointsPerUnit,
		&item.Unit,
		&item.SourceURL,
		&item.MonthlyLimit,
		&isMass,
		&isCritical,
		&isActive,
	)
	item.IsMass = isMass != 0
	item.IsCritical = isCritical != 0
	item.IsActive = isActive != 0
	return item, err
}

func listOfficerUsers(ctx context.Context, q queryer, districtID int) ([]*User, error) {
	sqlText := `SELECT id FROM users WHERE role = ?`
	args := []any{roleOfficer}
	if districtID > 0 {
		sqlText += ` AND district_id = ?`
		args = append(args, districtID)
	}
	sqlText += ` ORDER BY full_name`
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func envDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseFloat(value string, fallback float64) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(value, 64)
}

func parseBoolInt(value string) bool {
	value = strings.TrimSpace(value)
	return value == "1" || strings.EqualFold(value, "true")
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func validateMonth(month string) error {
	if len(month) != 7 || month[4] != '-' {
		return errors.New("invalid month")
	}
	year, errYear := strconv.Atoi(month[:4])
	monthNum, errMonth := strconv.Atoi(month[5:])
	if errYear != nil || errMonth != nil || year < 1 || monthNum < 1 || monthNum > 12 {
		return errors.New("invalid month")
	}
	return nil
}

func monthFromDate(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return ""
}

func monthShift(base time.Time, offset int) string {
	year, month, _ := base.Date()
	total := year*12 + int(month) - 1 + offset
	newYear := total / 12
	newMonth := total%12 + 1
	return fmt.Sprintf("%04d-%02d", newYear, newMonth)
}

func nowDB() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
