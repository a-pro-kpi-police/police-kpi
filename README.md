# KPI — Отдел полиции (KPI Dashboard for Police Department)

Веб-приложение для оценки эффективности сотрудников полиции. Сотрудники
вносят записи о выполненной работе (KPI), а начальник отдела подтверждает или
отклоняет их. Дашборд показывает агрегированные показатели эффективности.

- **Backend:** Python (FastAPI + SQLAlchemy + SQLite)
- **Frontend:** React (Vite) + Recharts
- **Языки интерфейса:** Русский / Қазақша (переключатель RU / KZ)
- **Роли:** `officer` (вносит записи) и `head` (подтверждает)

## Структура колонок записи KPI
Дата · Месяц · ФИО сотрудника · Код KPI · Показатель · Блок · НПА/статья ·
Балл за 1 ед. · Коэфф. участка · Сырые баллы · Итоговые баллы · Статус · Комментарии

Расчёт: **Сырые баллы** = Балл за 1 ед. × Количество.
**Итоговые баллы** = Сырые баллы × Коэфф. участка.

## Запуск

Требуется Docker с поддержкой `docker compose`.

### 1. Все сервисы
```bash
docker compose up --build
```
Откройте http://localhost:5173.

### 2. Только backend
```bash
docker compose up --build backend
```
Backend доступен на http://localhost:8000.

### 3. Frontend + backend
```bash
docker compose up --build frontend
```
Vite проксирует `/api` на backend-контейнер.

При первом запуске база создаётся и заполняется примерами автоматически.
Данные SQLite хранятся в Docker volume `backend_data`.

### Полезные команды
```bash
docker compose down          # остановить сервисы
docker compose down -v       # остановить и удалить volumes (сбросить БД)
docker compose logs -f       # смотреть логи
```

## Тестовые учётные записи
| Логин   | Пароль   | Роль              |
|---------|----------|-------------------|
| head    | head123  | Начальник отдела  |
| aibek   | pass123  | Сотрудник         |
| dana    | pass123  | Сотрудник         |
| ruslan  | pass123  | Сотрудник         |
| aliya   | pass123  | Сотрудник         |

## API
- `POST /api/auth/login` — вход (form: username, password)
- `GET  /api/auth/me` — текущий пользователь
- `GET  /api/catalog` — справочник показателей KPI
- `GET  /api/entries` — записи (сотрудник видит свои, начальник — все)
- `POST /api/entries` — создать запись (сотрудник)
- `PATCH /api/entries/{id}/status` — подтвердить/отклонить (начальник)
- `GET  /api/dashboard` — агрегаты для дашборда
- `GET  /api/officers` — рейтинг (через /dashboard)