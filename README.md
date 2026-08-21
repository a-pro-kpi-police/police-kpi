# KPI — отдел полиции

Веб-приложение для учёта и оценки KPI сотрудников полиции. Сотрудники
добавляют результаты работы и подтверждающие файлы, а начальник отдела
проверяет записи, назначает цели и отслеживает эффективность подразделения.

## Возможности

- личные кабинеты начальника отдела и сотрудников;
- создание, подтверждение и отклонение записей KPI;
- вложения к записям до 20 МБ;
- индивидуальные и групповые цели с отображением прогресса;
- дашборд, рейтинг сотрудников и фильтры по периоду и участку;
- системные уведомления об изменениях записей и целей;
- личные сообщения между начальником и сотрудниками с привязкой к KPI;
- русский и казахский языки интерфейса;
- светлая и тёмная темы.

## Стек

- **Backend:** Go 1.24, net/http, PostgreSQL
- **Frontend:** React 18, Vite, React Router, Axios, Recharts
- **Запуск:** Docker Compose или локальная среда разработки

## Быстрый запуск через Docker

Понадобятся Docker и плагин `docker compose`.

```bash
docker compose up --build
```

После запуска доступны:

- приложение — <http://localhost:5173>;
- API — <http://localhost:8000>;

Frontend проксирует запросы `/api` на backend-контейнер. PostgreSQL хранит
данные в Docker volume `postgres_data`, загруженные файлы сохраняются в
`backend_uploads`, а зависимости frontend — в `frontend_node_modules`.

Полезные команды:

```bash
docker compose logs -f
docker compose down
docker compose down -v  # удалить volumes и создать БД заново при следующем запуске
```

> `docker compose down -v` безвозвратно удаляет данные контейнерной БД и
> загруженные в неё вложения.

## Деплой

Для VPS/Cloud Server на PS.kz подготовлен отдельный production compose:

```bash
cp .env.ps.kz.example .env
docker compose -f docker-compose.prod.yml up -d --build
```

Подробная инструкция находится в [`docs/deploy-ps-kz.md`](docs/deploy-ps-kz.md).

## Тестовые учётные записи

Docker-конфигурация импортирует снимок данных за июль 2026 года.

| Логин | Пароль | Роль |
|---|---|---|
| `KPI_DEBUG` | Enable debug logs (`1`, `true`, `yes`, `on`) | off |
| `admin` | значение `KPI_ADMIN_PASSWORD` | Администратор |
| `head` | `head123` | Начальник отдела |
| `officer001` … `officer152` | `pass123` | Импортированные сотрудники |

При локальном запуске с новой БД и без режима месячного импорта создаются
демонстрационные пользователи:

| Логин | Пароль | Роль |
|---|---|---|
| `admin` | `admin123` | Администратор |
| `head` | `head123` | Начальник отдела |
| `officer001` … `officer020` | `pass123` | Сотрудники |

Учётные данные предназначены только для разработки. Перед публикацией
приложения задайте собственные пароли и значение `KPI_SECRET`.

## Локальная разработка

### Backend

```bash
cd backend-golang
go mod tidy
go run .
```

Если `KPI_DATABASE_URL` не задан, используется PostgreSQL `postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable`.
При первом запуске пустая БД автоматически заполняется демонстрационными
данными.

### Frontend

В отдельном терминале:

```bash
cd frontend
npm ci
npm run dev
```

Vite по умолчанию отправляет запросы `/api` на
`http://localhost:8000`. Другой адрес можно задать через
`VITE_API_PROXY_TARGET`.

## Конфигурация backend

| Переменная | Назначение | Значение по умолчанию |
|---|---|---|
| `KPI_DATABASE_URL` | URL подключения PostgreSQL | `postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable` |
| `KPI_SECRET` | Секрет подписи JWT | тестовое значение в исходном коде |
| `KPI_UPLOAD_DIR` | Каталог для вложений | `backend-golang/uploads` |
| `KPI_SEED_MODE` | Режим начального заполнения | демонстрационные данные |
| `KPI_MONTHLY_IMPORT_PATH` | Путь к CSV месячного импорта | `backend-golang/import_data/monthly_kpi_2026_07.csv` |
| `KPI_ADMIN_PASSWORD` | Пароль системного администратора | `admin123` |
| `KPI_HEAD_PASSWORD` | Пароль начальника при месячном импорте | `head123` |
| `KPI_IMPORT_DEFAULT_PASSWORD` | Пароль импортированных сотрудников | `pass123` |

Чтобы локально запустить импорт июльского снимка в отдельную БД:

```bash
cd backend-golang
KPI_DATABASE_URL=postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable \
KPI_SEED_MODE=monthly_import \
go run .
```

## Месячный импорт

Docker использует отдельную базу `/data/kpi_july_2026.db`; существующий файл
`/data/kpi.db` не удаляется и не перезаписывается. При первом запуске
импортируется `backend-golang/import_data/monthly_kpi_2026_07.csv`:

- 152 сотрудника;
- 134 показателя KPI;
- 20 368 значений «сотрудник × KPI», включая нулевые;
- даты записей распределены с 1 по 29 июля 2026 года.

Импорт идемпотентен: повторная обработка того же файла определяется по его
SHA-256 и не создаёт дубликаты.

На macOS CSV и каталог KPI можно повторно получить из исходного Numbers-файла:

```bash
osascript -l JavaScript scripts/export_monthly_numbers.js \
  ./KPI_*.numbers \
  backend-golang/import_data/monthly_kpi_2026_07.csv \
  backend-golang/kpi_catalog.csv
```

Подробности формата снимка находятся в
[`backend-golang/import_data/README.md`](backend-golang/import_data/README.md).

## Массовое создание кабинетов сотрудников

Когда передадут список сотрудников, подготовьте CSV по шаблону
`backend-golang/import_data/users_template.csv`:

```csv
employee_number,full_name,rank,district,monthly_norm
1001,Иванов Иван Иванович,Лейтенант полиции,Батальон патрульной полиции,100
```

Для ФИО в формате `Фамилия Имя Отчество` запуск:

```bash
cd backend-golang
go run ./cmd/import-users \
  -input import_data/users.csv \
  -output import_data/users_accounts.csv
```

Логин генерируется латиницей по правилу `имя.первая_буква_фамилии`, например
`ivan.i`. Если логин совпадает с существующим, добавляется индекс: `ivan.i2`,
`ivan.i3`. Пароль создаётся как логин плюс случайные цифры. В базе хранится
только хэш пароля, а открытые пароли записываются в output CSV.

Полезные флаги:

- `-dry-run` — проверить импорт без сохранения в БД.
- `-reset-existing-passwords` — пересоздать пароли уже найденным сотрудникам.
- `-name-order ifo` — если ФИО в файле идёт как `Имя Фамилия Отчество`.

Пароли из `-dry-run` не выдавайте сотрудникам: это только превью, изменения в
БД откатываются.

## Расчёт KPI

Для каждого подтверждённого показателя:

1. Полное заявленное количество сохраняется в статистике.
2. В расчёт попадает `MIN(количество, месячный лимит)`.
3. Балл умножается на коэффициент участка сотрудника.
4. Доля массовых показателей рассчитывается от положительных баллов после
   применения лимитов.
5. При доле массовых показателей свыше 40% применяется коэффициент `0,85`, а
   свыше 60% — `0,70`.
6. Подтверждённый критический показатель обнуляет результат сотрудника за
   месяц.
7. Эффективность сравнивается с месячной нормой 100 баллов, а не с суммой
   назначенных целей.

## API

Все рабочие маршруты, кроме входа, требуют Bearer JWT. JSON-контракты Go API
сохранены совместимыми с фронтендом.

| Метод и маршрут | Назначение |
|---|---|
| `POST /api/auth/login` | Вход по логину и паролю |
| `GET /api/auth/me` | Текущий пользователь |
| `GET /api/catalog` | Активный справочник KPI |
| `GET /api/districts` | Список участков |
| `GET /api/months` | Доступные отчётные месяцы |
| `GET /api/officers` | Сотрудники и их рейтинг, только для начальника |
| `GET /api/entries` | Записи KPI с фильтрацией, сортировкой и пагинацией |
| `POST /api/entries` | Создать запись |
| `POST /api/entries/with-attachment` | Создать запись с вложением |
| `GET /api/entries/{id}/attachment` | Скачать вложение |
| `PATCH /api/entries/{id}/status` | Изменить статус записи, только для начальника |
| `GET/PUT /api/targets` | Индивидуальные цели |
| `GET/PUT /api/group-targets` | Групповые цели участка |
| `GET /api/target-progress` | Прогресс по целям сотрудника |
| `GET /api/target-overview` | Сводка выполнения целей |
| `GET /api/dashboard` | Агрегаты дашборда, только для начальника |
| `GET/PATCH/POST /api/notifications...` | Получение и прочтение уведомлений |
| `GET/POST/DELETE /api/messenger...` | Контакты, сообщения и обсуждение KPI |

## Тесты и сборка

```bash
cd backend-golang
go test ./...
```

```bash
cd frontend
npm run build
```

## Структура проекта

```text
.
├── backend-golang/
│   ├── import_data/      # CSV-снимок месячного импорта
│   ├── kpi_catalog.csv   # активный справочник показателей
│   ├── main.go           # Go HTTP API
│   ├── models.go         # Go-модели и JSON-контракты
│   ├── handlers_*.go     # REST API
│   ├── scoring.go        # расчёт KPI
│   └── seed.go           # создание, миграция и заполнение БД
├── frontend/
│   └── src/              # React-приложение
├── scripts/
│   └── export_monthly_numbers.js
└── docker-compose.yml
```
