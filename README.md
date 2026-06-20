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

### 1. Backend (порт 8000)
```bash
./run_backend.sh
# или вручную:
cd backend && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt
.venv/bin/uvicorn main:app --reload --port 8000
```
При первом запуске база `kpi.db` создаётся и заполняется примерами автоматически.

### 2. Frontend (порт 5173) — требуется Node.js 18+
```bash
./run_frontend.sh
# или вручную:
cd frontend && npm install && npm run dev
```
Откройте http://localhost:5173 (Vite проксирует `/api` на backend:8000).

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
