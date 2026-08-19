package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoginBaseUsesFirstNameAndSurnameInitial(t *testing.T) {
	employee := employeeInput{
		FullName:  "Иванов Айбек Маратулы",
		FirstName: "Айбек",
		LastName:  "Иванов",
	}

	if got := loginBase(employee); got != "aibek.i" {
		t.Fatalf("loginBase() = %q, want %q", got, "aibek.i")
	}
}

func TestUniqueUsernameIndexesCollisions(t *testing.T) {
	taken := map[string]bool{
		"aibek.i":  true,
		"aibek.i2": true,
	}

	if got := uniqueUsername("aibek.i", taken); got != "aibek.i3" {
		t.Fatalf("uniqueUsername() = %q, want %q", got, "aibek.i3")
	}
	if !taken["aibek.i3"] {
		t.Fatal("uniqueUsername() did not reserve generated username")
	}
}

func TestReadEmployeesSupportsRussianSemicolonCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "employees.csv")
	data := "табельный номер;фио;звание;участок;норма;роль\n" +
		"001;Иванов Айбек Маратулы;Лейтенант полиции;Батальон;110,5;head\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	employees, err := readEmployees(path, "fio", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(employees) != 1 {
		t.Fatalf("len(employees) = %d, want 1", len(employees))
	}
	employee := employees[0]
	if employee.EmployeeNumber != "001" {
		t.Fatalf("EmployeeNumber = %q, want 001", employee.EmployeeNumber)
	}
	if employee.FirstName != "Айбек" || employee.LastName != "Иванов" {
		t.Fatalf("parsed name = (%q, %q), want (Айбек, Иванов)", employee.FirstName, employee.LastName)
	}
	if employee.MonthlyNorm != 110.5 {
		t.Fatalf("MonthlyNorm = %v, want 110.5", employee.MonthlyNorm)
	}
	if employee.Role != roleHead {
		t.Fatalf("Role = %q, want head", employee.Role)
	}
}

func TestReadEmployeesRejectsDuplicateEmployeeNumbers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "employees.csv")
	data := "employee_number,full_name\n" +
		"1,Иванов Айбек Маратулы\n" +
		"1,Петров Айбек Маратулы\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := readEmployees(path, "fio", 100)
	if err == nil || !strings.Contains(err.Error(), "duplicate employee_number") {
		t.Fatalf("readEmployees() error = %v, want duplicate employee_number", err)
	}
}

func TestRunCreatesUsersAndCredentialCSV(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kpi.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE districts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			coeff REAL NOT NULL DEFAULT 1.0
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT NOT NULL,
			rank TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'officer',
			district_id INTEGER REFERENCES districts(id),
			source_employee_number TEXT UNIQUE,
			monthly_norm REAL NOT NULL DEFAULT 100.0
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	inputPath := filepath.Join(dir, "users.csv")
	outputPath := filepath.Join(dir, "users_accounts.csv")
	data := "employee_number,full_name,rank,district,role\n" +
		"1001,Иванов Иван Иванович,Лейтенант полиции,Батальон,head\n" +
		"1002,Исаев Иван Иванович,Лейтенант полиции,Батальон,officer\n"
	if err := os.WriteFile(inputPath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	err = run(context.Background(), options{
		input:              inputPath,
		output:             outputPath,
		databaseURL:        "sqlite:///" + dbPath,
		nameOrder:          "fio",
		passwordDigits:     2,
		defaultMonthlyNorm: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT username, role FROM users ORDER BY username`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var usernames []string
	roles := map[string]string{}
	for rows.Next() {
		var username, role string
		if err := rows.Scan(&username, &role); err != nil {
			t.Fatal(err)
		}
		usernames = append(usernames, username)
		roles[username] = role
	}
	if strings.Join(usernames, ",") != "ivan.i,ivan.i2" {
		t.Fatalf("usernames = %v, want [ivan.i ivan.i2]", usernames)
	}
	if roles["ivan.i"] != roleHead || roles["ivan.i2"] != roleOfficer {
		t.Fatalf("roles = %v, want ivan.i=head and ivan.i2=officer", roles)
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outputFile.Close()
	records, err := csv.NewReader(outputFile).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("credential rows = %d, want 3 including header", len(records))
	}
	if records[1][2] != "ivan.i" || !strings.HasPrefix(records[1][3], "ivan.i") || len(records[1][3]) != len("ivan.i")+2 {
		t.Fatalf("first credential row = %v", records[1])
	}
	if records[2][2] != "ivan.i2" || !strings.HasPrefix(records[2][3], "ivan.i2") || len(records[2][3]) != len("ivan.i2")+2 {
		t.Fatalf("second credential row = %v", records[2])
	}
}

func TestRunReplaceDataPurgesOperationalTables(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kpi.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE districts (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE NOT NULL, coeff REAL NOT NULL DEFAULT 1.0)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT NOT NULL,
			rank TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'officer',
			district_id INTEGER REFERENCES districts(id),
			source_employee_number TEXT UNIQUE,
			monthly_norm REAL NOT NULL DEFAULT 100.0
		)`,
		`CREATE TABLE direct_messages (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE app_notifications (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE kpi_entries (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE kpi_targets (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE import_batches (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`INSERT INTO districts(name, coeff) VALUES('Demo', 1)`,
		`INSERT INTO users(username, password_hash, full_name, rank, role, district_id, source_employee_number, monthly_norm)
		 VALUES('old', 'hash', 'Old User', '', 'officer', 1, 'old-1', 100)`,
		`INSERT INTO direct_messages DEFAULT VALUES`,
		`INSERT INTO app_notifications DEFAULT VALUES`,
		`INSERT INTO kpi_entries DEFAULT VALUES`,
		`INSERT INTO kpi_targets DEFAULT VALUES`,
		`INSERT INTO import_batches DEFAULT VALUES`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	inputPath := filepath.Join(dir, "users.csv")
	outputPath := filepath.Join(dir, "users_accounts.csv")
	data := "employee_number,full_name,rank,district,role\n" +
		"1001,Иванов Иван Иванович,Лейтенант полиции,1 Рота,head\n"
	if err := os.WriteFile(inputPath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	err = run(context.Background(), options{
		input:              inputPath,
		output:             outputPath,
		databaseURL:        "sqlite:///" + dbPath,
		nameOrder:          "fio",
		passwordDigits:     2,
		defaultMonthlyNorm: 100,
		replaceData:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for table, want := range map[string]int{
		"users":             1,
		"districts":         1,
		"direct_messages":   0,
		"app_notifications": 0,
		"kpi_entries":       0,
		"kpi_targets":       0,
		"import_batches":    0,
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
}
