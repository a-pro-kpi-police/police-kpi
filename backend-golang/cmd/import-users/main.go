package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

const (
	dialectPostgres = "postgres"
	dialectSQLite   = "sqlite"
	roleHead        = "head"
	roleOfficer     = "officer"
)

type options struct {
	input                  string
	output                 string
	databaseURL            string
	nameOrder              string
	passwordDigits         int
	defaultMonthlyNorm     float64
	dryRun                 bool
	resetExistingPasswords bool
	replaceData            bool
}

type txStore struct {
	tx      *sql.Tx
	dialect string
}

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type employeeInput struct {
	EmployeeNumber string
	FullName       string
	FirstName      string
	LastName       string
	Rank           string
	District       string
	Role           string
	MonthlyNorm    float64
}

type existingUser struct {
	ID       int
	Username string
}

type accountResult struct {
	EmployeeNumber string
	FullName       string
	Username       string
	Password       string
	Action         string
	Notes          string
}

func main() {
	var opts options
	flag.StringVar(&opts.input, "input", "", "CSV file with employees")
	flag.StringVar(&opts.output, "output", "", "CSV file for generated credentials")
	flag.StringVar(&opts.databaseURL, "database-url", envDefault("KPI_DATABASE_URL", "postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable"), "database URL")
	flag.StringVar(&opts.nameOrder, "name-order", "fio", "full_name order: fio means Фамилия Имя Отчество, ifo means Имя Фамилия Отчество")
	flag.IntVar(&opts.passwordDigits, "password-digits", 4, "number of random digits appended to the login")
	flag.Float64Var(&opts.defaultMonthlyNorm, "default-monthly-norm", 100, "monthly_norm for rows without this column")
	flag.BoolVar(&opts.dryRun, "dry-run", false, "calculate accounts and roll back database changes")
	flag.BoolVar(&opts.resetExistingPasswords, "reset-existing-passwords", false, "generate new passwords for users found by employee number")
	flag.BoolVar(&opts.replaceData, "replace-data", false, "delete existing users, KPI entries, targets, messages, notifications, and import batches before import; preserves KPI catalog")
	flag.Parse()

	if opts.input == "" {
		fmt.Fprintln(os.Stderr, "missing required -input")
		flag.Usage()
		os.Exit(2)
	}
	if opts.output == "" {
		opts.output = defaultOutputPath(opts.input)
	}
	if opts.passwordDigits < 1 {
		fmt.Fprintln(os.Stderr, "-password-digits must be at least 1")
		os.Exit(2)
	}
	if opts.nameOrder != "fio" && opts.nameOrder != "ifo" {
		fmt.Fprintln(os.Stderr, "-name-order must be fio or ifo")
		os.Exit(2)
	}

	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options) error {
	employees, err := readEmployees(opts.input, opts.nameOrder, opts.defaultMonthlyNorm)
	if err != nil {
		return err
	}
	if len(employees) == 0 {
		return errors.New("employee CSV has no rows")
	}

	db, dialect, err := openDatabase(opts.databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := txStore{tx: tx, dialect: dialect}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if opts.replaceData {
		if err := replaceOperationalData(ctx, q); err != nil {
			return err
		}
	}

	taken, err := existingUsernames(ctx, q)
	if err != nil {
		return err
	}

	results := make([]accountResult, 0, len(employees))
	for _, employee := range employees {
		result, err := importEmployee(ctx, q, employee, taken, opts)
		if err != nil {
			return fmt.Errorf("%s: %w", employee.FullName, err)
		}
		results = append(results, result)
	}

	if opts.dryRun {
		if err := tx.Rollback(); err != nil {
			return err
		}
		committed = true
	} else {
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
	}

	if err := writeAccountResults(opts.output, results); err != nil {
		return err
	}

	created, updated, reset := summarize(results)
	mode := "created/updated"
	if opts.dryRun {
		mode = "dry-run previewed"
	}
	fmt.Printf("%s %d accounts: %d new, %d updated, %d passwords reset\n", mode, len(results), created, updated, reset)
	fmt.Printf("credentials written to %s\n", opts.output)
	if opts.dryRun {
		fmt.Println("dry-run: database changes were rolled back; generated passwords are preview only")
	}
	return nil
}

func importEmployee(ctx context.Context, q txStore, employee employeeInput, taken map[string]bool, opts options) (accountResult, error) {
	districtID, districtCreated, err := ensureDistrict(ctx, q, employee.District)
	if err != nil {
		return accountResult{}, err
	}

	if employee.EmployeeNumber != "" {
		if existing, found, err := userByEmployeeNumber(ctx, q, employee.EmployeeNumber); err != nil {
			return accountResult{}, err
		} else if found {
			result := accountResult{
				EmployeeNumber: employee.EmployeeNumber,
				FullName:       employee.FullName,
				Username:       existing.Username,
				Action:         actionName(opts.dryRun, "updated"),
			}
			passwordHash := ""
			if opts.resetExistingPasswords {
				password, err := generatedPassword(existing.Username, opts.passwordDigits)
				if err != nil {
					return accountResult{}, err
				}
				hash, err := hashPassword(password)
				if err != nil {
					return accountResult{}, err
				}
				passwordHash = hash
				result.Password = password
				result.Action = actionName(opts.dryRun, "password_reset")
			}
			if districtCreated {
				result.Notes = "district created"
			}
			if err := updateUser(ctx, q, existing.ID, employee, districtID, passwordHash); err != nil {
				return accountResult{}, err
			}
			return result, nil
		}
	}

	username := uniqueUsername(loginBase(employee), taken)
	password, err := generatedPassword(username, opts.passwordDigits)
	if err != nil {
		return accountResult{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return accountResult{}, err
	}
	if err := insertUser(ctx, q, employee, username, passwordHash, districtID); err != nil {
		return accountResult{}, err
	}

	notes := ""
	if districtCreated {
		notes = "district created"
	}
	return accountResult{
		EmployeeNumber: employee.EmployeeNumber,
		FullName:       employee.FullName,
		Username:       username,
		Password:       password,
		Action:         actionName(opts.dryRun, "created"),
		Notes:          notes,
	}, nil
}

func readEmployees(path, nameOrder string, defaultMonthlyNorm float64) ([]employeeInput, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimPrefix(string(raw), "\ufeff")
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	if detectSemicolon(text) {
		reader.Comma = ';'
	}

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV headers: %w", err)
	}
	indexes := headerIndexes(headers)
	if _, ok := indexes["full_name"]; !ok {
		if _, firstOK := indexes["first_name"]; !firstOK {
			return nil, errors.New("CSV must contain full_name/ФИО or first_name/last_name columns")
		}
		if _, lastOK := indexes["last_name"]; !lastOK {
			return nil, errors.New("CSV must contain full_name/ФИО or first_name/last_name columns")
		}
	}

	var employees []employeeInput
	seenEmployeeNumbers := map[string]int{}
	line := 1
	for {
		line++
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV line %d: %w", line, err)
		}
		if blankRecord(record) {
			continue
		}
		employee, err := employeeFromRecord(record, indexes, nameOrder, defaultMonthlyNorm)
		if err != nil {
			return nil, fmt.Errorf("CSV line %d: %w", line, err)
		}
		if employee.EmployeeNumber != "" {
			if previousLine, ok := seenEmployeeNumbers[employee.EmployeeNumber]; ok {
				return nil, fmt.Errorf("CSV line %d: duplicate employee_number %q, first seen on line %d", line, employee.EmployeeNumber, previousLine)
			}
			seenEmployeeNumbers[employee.EmployeeNumber] = line
		}
		employees = append(employees, employee)
	}
	return employees, nil
}

func employeeFromRecord(record []string, indexes map[string]int, nameOrder string, defaultMonthlyNorm float64) (employeeInput, error) {
	fullName := field(record, indexes, "full_name")
	firstName := field(record, indexes, "first_name")
	lastName := field(record, indexes, "last_name")
	if fullName == "" {
		fullName = strings.Join(nonEmpty(lastName, firstName, field(record, indexes, "middle_name")), " ")
	}
	if firstName == "" || lastName == "" {
		parsedFirst, parsedLast := parseFullName(fullName, nameOrder)
		if firstName == "" {
			firstName = parsedFirst
		}
		if lastName == "" {
			lastName = parsedLast
		}
	}
	if strings.TrimSpace(fullName) == "" {
		return employeeInput{}, errors.New("full_name is required")
	}
	if strings.TrimSpace(firstName) == "" {
		return employeeInput{}, errors.New("cannot detect first name")
	}
	if strings.TrimSpace(lastName) == "" {
		return employeeInput{}, errors.New("cannot detect last name")
	}

	monthlyNorm := defaultMonthlyNorm
	if rawNorm := field(record, indexes, "monthly_norm"); rawNorm != "" {
		value, err := parseFloat(rawNorm)
		if err != nil {
			return employeeInput{}, fmt.Errorf("monthly_norm: %w", err)
		}
		monthlyNorm = value
	}
	role, err := normalizeRole(field(record, indexes, "role"))
	if err != nil {
		return employeeInput{}, err
	}

	return employeeInput{
		EmployeeNumber: cleanEmployeeNumber(field(record, indexes, "employee_number")),
		FullName:       strings.TrimSpace(fullName),
		FirstName:      strings.TrimSpace(firstName),
		LastName:       strings.TrimSpace(lastName),
		Rank:           field(record, indexes, "rank"),
		District:       field(record, indexes, "district"),
		Role:           role,
		MonthlyNorm:    monthlyNorm,
	}, nil
}

func headerIndexes(headers []string) map[string]int {
	normalized := map[string]int{}
	for i, header := range headers {
		normalized[normalizeHeader(header)] = i
	}

	result := map[string]int{}
	for canonical, aliases := range headerAliases() {
		for _, alias := range aliases {
			if index, ok := normalized[normalizeHeader(alias)]; ok {
				result[canonical] = index
				break
			}
		}
	}
	return result
}

func headerAliases() map[string][]string {
	return map[string][]string{
		"employee_number": {
			"employee_number", "source_employee_number", "employee no", "id",
			"табельный", "табельный номер", "номер", "номер сотрудника",
		},
		"full_name": {
			"full_name", "full name", "fio", "фио", "сотрудник",
			"фамилия имя отчество",
		},
		"last_name": {
			"last_name", "last name", "surname", "фамилия",
		},
		"first_name": {
			"first_name", "first name", "name", "имя",
		},
		"middle_name": {
			"middle_name", "middle name", "patronymic", "отчество",
		},
		"rank": {
			"rank", "position", "звание", "должность",
		},
		"district": {
			"district", "district_name", "unit", "group",
			"участок", "группа", "подразделение", "батальон",
		},
		"monthly_norm": {
			"monthly_norm", "norm", "target_score",
			"норма", "месячная норма", "целевой балл", "цель",
		},
		"role": {
			"role", "account_role", "user_role",
			"роль", "рөл", "тип аккаунта", "права",
		},
	}
}

func normalizeHeader(value string) string {
	value = strings.TrimPrefix(value, "\ufeff")
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_")
	return replacer.Replace(value)
}

func field(record []string, indexes map[string]int, key string) string {
	index, ok := indexes[key]
	if !ok || index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func detectSemicolon(text string) bool {
	firstLine := text
	if index := strings.IndexAny(text, "\r\n"); index >= 0 {
		firstLine = text[:index]
	}
	return strings.Count(firstLine, ";") > strings.Count(firstLine, ",")
}

func blankRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseFullName(fullName, nameOrder string) (firstName, lastName string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], parts[0]
	}
	if nameOrder == "ifo" {
		return parts[0], parts[1]
	}
	return parts[1], parts[0]
}

func normalizeRole(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", roleOfficer, "employee", "сотрудник", "қызметкер":
		return roleOfficer, nil
	case roleHead, "commander", "chief", "manager", "начальник", "командир", "глава", "рота командирі":
		return roleHead, nil
	default:
		return "", fmt.Errorf("invalid role %q: expected head or officer", value)
	}
}

func loginBase(employee employeeInput) string {
	first := sanitizeLoginPart(transliterate(employee.FirstName))
	last := sanitizeLoginPart(transliterate(employee.LastName))
	if first == "" {
		first = "user"
	}
	initial := firstRune(last)
	if initial == "" {
		initial = "x"
	}
	return first + "." + initial
}

func uniqueUsername(base string, taken map[string]bool) string {
	username := base
	for index := 2; taken[username]; index++ {
		username = fmt.Sprintf("%s%d", base, index)
	}
	taken[username] = true
	return username
}

func generatedPassword(username string, digits int) (string, error) {
	var builder strings.Builder
	builder.WriteString(username)
	for i := 0; i < digits; i++ {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteString(value.String())
	}
	return builder.String(), nil
}

func transliterate(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if r <= unicode.MaxASCII {
			builder.WriteRune(r)
			continue
		}
		if replacement, ok := transliterationTable()[r]; ok {
			builder.WriteString(replacement)
		}
	}
	return builder.String()
}

func transliterationTable() map[rune]string {
	return map[rune]string{
		'а': "a", 'ә': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "g",
		'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "i", 'к': "k", 'қ': "k", 'л': "l", 'м': "m", 'н': "n",
		'ң': "n", 'о': "o", 'ө': "o", 'п': "p", 'р': "r", 'с': "s",
		'т': "t", 'у': "u", 'ұ': "u", 'ү': "u", 'ф': "f", 'х': "h",
		'һ': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
		'ъ': "", 'ы': "y", 'і': "i", 'ь': "", 'э': "e", 'ю': "yu",
		'я': "ya",
	}
}

func sanitizeLoginPart(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func firstRune(value string) string {
	r, size := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError && size == 0 {
		return ""
	}
	return string(r)
}

func openDatabase(databaseURL string) (*sql.DB, string, error) {
	dialect := dialectPostgres
	dsn := databaseURL
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"), strings.HasPrefix(databaseURL, "postgresql://"):
		dialect = dialectPostgres
	case strings.HasPrefix(databaseURL, "sqlite:///"):
		dialect = dialectSQLite
		dsn = strings.TrimPrefix(databaseURL, "sqlite:///")
	case strings.HasPrefix(databaseURL, "sqlite://"):
		dialect = dialectSQLite
		dsn = strings.TrimPrefix(databaseURL, "sqlite://")
	case strings.HasSuffix(databaseURL, ".db"), strings.HasSuffix(databaseURL, ".sqlite3"):
		dialect = dialectSQLite
	default:
		return nil, "", fmt.Errorf("unsupported KPI_DATABASE_URL: %s", databaseURL)
	}
	db, err := sql.Open(dialect, dsn)
	if err != nil {
		return nil, "", err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, "", err
	}
	return db, dialect, nil
}

func existingUsernames(ctx context.Context, q queryer) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT username FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]bool{}
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		result[username] = true
	}
	return result, rows.Err()
}

func userByEmployeeNumber(ctx context.Context, q queryer, employeeNumber string) (existingUser, bool, error) {
	var user existingUser
	err := q.QueryRowContext(ctx, `SELECT id, username FROM users WHERE source_employee_number = ?`, employeeNumber).Scan(&user.ID, &user.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return existingUser{}, false, nil
	}
	if err != nil {
		return existingUser{}, false, err
	}
	return user, true, nil
}

func ensureDistrict(ctx context.Context, q queryer, district string) (*int, bool, error) {
	district = strings.TrimSpace(district)
	if district == "" {
		return nil, false, nil
	}
	var id int
	err := q.QueryRowContext(ctx, `SELECT id FROM districts WHERE name = ?`, district).Scan(&id)
	if err == nil {
		return &id, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if err := q.QueryRowContext(ctx, `INSERT INTO districts(name, coeff) VALUES (?, 1.0) RETURNING id`, district).Scan(&id); err != nil {
		return nil, false, err
	}
	return &id, true, nil
}

func replaceOperationalData(ctx context.Context, q queryer) error {
	statements := []string{
		`DELETE FROM direct_messages`,
		`DELETE FROM app_notifications`,
		`DELETE FROM kpi_entries`,
		`DELETE FROM kpi_targets`,
		`DELETE FROM import_batches`,
		`DELETE FROM users`,
		`DELETE FROM districts`,
	}
	for _, statement := range statements {
		if _, err := q.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertUser(ctx context.Context, q queryer, employee employeeInput, username, passwordHash string, districtID *int) error {
	_, err := q.ExecContext(ctx, `INSERT INTO users
		(username, password_hash, full_name, rank, role, district_id, source_employee_number, monthly_norm)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		username,
		passwordHash,
		employee.FullName,
		employee.Rank,
		employee.Role,
		nullableInt(districtID),
		nullableString(employee.EmployeeNumber),
		employee.MonthlyNorm,
	)
	return err
}

func updateUser(ctx context.Context, q queryer, userID int, employee employeeInput, districtID *int, passwordHash string) error {
	if passwordHash != "" {
		_, err := q.ExecContext(ctx, `UPDATE users
			SET password_hash = ?, full_name = ?, rank = ?, role = ?, district_id = ?, monthly_norm = ?
			WHERE id = ?`,
			passwordHash,
			employee.FullName,
			employee.Rank,
			employee.Role,
			nullableInt(districtID),
			employee.MonthlyNorm,
			userID,
		)
		return err
	}
	_, err := q.ExecContext(ctx, `UPDATE users
		SET full_name = ?, rank = ?, role = ?, district_id = ?, monthly_norm = ?
		WHERE id = ?`,
		employee.FullName,
		employee.Rank,
		employee.Role,
		nullableInt(districtID),
		employee.MonthlyNorm,
		userID,
	)
	return err
}

func writeAccountResults(path string, results []accountResult) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString("\ufeff"); err != nil {
		return err
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"employee_number", "full_name", "username", "password", "action", "notes"}); err != nil {
		return err
	}
	for _, result := range results {
		if err := writer.Write([]string{
			result.EmployeeNumber,
			result.FullName,
			result.Username,
			result.Password,
			result.Action,
			result.Notes,
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func (q txStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.tx.ExecContext(ctx, rebind(query, q.dialect), args...)
}

func (q txStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return q.tx.QueryContext(ctx, rebind(query, q.dialect), args...)
}

func (q txStore) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return q.tx.QueryRowContext(ctx, rebind(query, q.dialect), args...)
}

func rebind(query string, dialect string) string {
	if dialect != dialectPostgres {
		return query
	}
	var builder strings.Builder
	builder.Grow(len(query) + 8)
	inSingleQuote := false
	index := 1
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			builder.WriteByte(ch)
			if inSingleQuote && i+1 < len(query) && query[i+1] == '\'' {
				i++
				builder.WriteByte(query[i])
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}
		if ch == '?' && !inSingleQuote {
			builder.WriteString("$")
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2Key([]byte(password), salt, 100_000, 32, sha256.New)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(key), nil
}

func pbkdf2Key(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var output []byte
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		_, _ = prf.Write(salt)
		_, _ = prf.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := prf.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			prf.Reset()
			_, _ = prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		output = append(output, t...)
	}
	return output[:keyLen]
}

func parseFloat(value string) (float64, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	return strconv.ParseFloat(value, 64)
}

func cleanEmployeeNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, ",", ".")
	if strings.HasSuffix(value, ".0") {
		prefix := strings.TrimSuffix(value, ".0")
		if prefix != "" && allDigits(prefix) {
			return prefix
		}
	}
	return value
}

func allDigits(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
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

func actionName(dryRun bool, action string) string {
	if dryRun {
		return "would_" + action
	}
	return action
}

func summarize(results []accountResult) (created, updated, reset int) {
	for _, result := range results {
		action := strings.TrimPrefix(result.Action, "would_")
		switch action {
		case "created":
			created++
		case "updated":
			updated++
		case "password_reset":
			updated++
			reset++
		}
	}
	return created, updated, reset
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func defaultOutputPath(input string) string {
	ext := filepath.Ext(input)
	base := strings.TrimSuffix(input, ext)
	if base == "" {
		base = "generated_user"
	}
	return base + "_accounts.csv"
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
