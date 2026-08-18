# KPI backend in Go

Go backend for the KPI application. It keeps the same `/api` routes,
PostgreSQL schema, JWT login, KPI scoring rules, CSV catalog sync, monthly CSV
import, attachments, notifications, targets, dashboard, and messenger.

Run locally with PostgreSQL running on `localhost:5432`:

```bash
go mod tidy
go test ./...
go run .
```

Environment variables:

- `KPI_DATABASE_URL`, default `postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable`
- `KPI_DEBUG=1` enables debug logs
- `KPI_SECRET`
- `KPI_UPLOAD_DIR`, default `./uploads`
- `KPI_SEED_MODE=monthly_import`
- `KPI_MONTHLY_IMPORT_PATH`, default `import_data/monthly_kpi_2026_07.csv`
- `KPI_HEAD_PASSWORD`, default `head123`
- `KPI_IMPORT_DEFAULT_PASSWORD`, default `pass123`

SQLite URLs such as `sqlite:///./kpi.db` are still accepted for lightweight
local tests, but Docker Compose uses PostgreSQL.

## Bulk employee account import

Prepare a CSV like `import_data/users_template.csv`:

```csv
employee_number,full_name,rank,district,monthly_norm
1001,Иванов Иван Иванович,Лейтенант полиции,Батальон патрульной полиции,100
```

Then run:

```bash
go run ./cmd/import-users \
  -input import_data/users.csv \
  -output import_data/users_accounts.csv
```

The importer creates officer accounts and writes credentials to the output CSV.
Login format is `firstname.surnameInitial` in Latin letters, for example
`ivan.i`. If the login already exists, the importer appends an index:
`ivan.i2`, `ivan.i3`, and so on. Passwords use the generated login plus random
digits. Existing users are matched by `employee_number` /
`source_employee_number`; by default their profile is updated without resetting
the password.

Useful flags:

- `-dry-run` previews changes and rolls back database writes.
- `-reset-existing-passwords` creates new passwords for matched existing users.
- `-name-order fio` means `full_name` is `Фамилия Имя Отчество` (default).
- `-name-order ifo` means `full_name` is `Имя Фамилия Отчество`.

Do not hand out credentials produced by `-dry-run`; they are only a preview.
