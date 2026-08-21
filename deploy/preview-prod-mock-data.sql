-- Preview auto-seeded mock users and KPI rows before cleanup.
-- Run against the production PostgreSQL database before deploy/cleanup-prod-mock-data.sql.

BEGIN;

CREATE TEMP TABLE cleanup_mock_users(id INTEGER PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE cleanup_mock_entries(id INTEGER PRIMARY KEY) ON COMMIT DROP;

INSERT INTO cleanup_mock_users(id)
SELECT DISTINCT u.id
FROM users u
WHERE u.role = 'officer'
  AND (
    EXISTS (
      SELECT 1
      FROM kpi_entries e
      WHERE e.officer_id = u.id
        AND e.source_key LIKE 'numbers:%'
    )
    OR (
      u.username ~ '^officer[0-9]{3}$'
      AND u.full_name ~ '^Сотрудник [0-9]{3}$'
    )
    OR (
      u.username ~ '^officer[0-9]{3}$'
      AND COALESCE(u.source_employee_number, '') ~ '^[0-9]+$'
      AND EXISTS (
        SELECT 1
        FROM import_batches b
        WHERE b.source_name = 'monthly_kpi_2026_07.csv'
           OR b.source_sheet = 'Месячная_таблица-1'
      )
    )
  );

INSERT INTO cleanup_mock_entries(id)
SELECT DISTINCT e.id
FROM kpi_entries e
WHERE e.officer_id IN (SELECT id FROM cleanup_mock_users)
   OR e.source_key LIKE 'numbers:%';

SELECT
  (SELECT COUNT(*) FROM cleanup_mock_users) AS mock_users,
  (SELECT COUNT(*) FROM cleanup_mock_entries) AS mock_entries,
  (SELECT COUNT(*) FROM kpi_targets WHERE officer_id IN (SELECT id FROM cleanup_mock_users)) AS mock_targets,
  (SELECT COUNT(*) FROM direct_messages WHERE sender_id IN (SELECT id FROM cleanup_mock_users)
     OR recipient_id IN (SELECT id FROM cleanup_mock_users)
     OR entry_id IN (SELECT id FROM cleanup_mock_entries)) AS mock_messages,
  (SELECT COUNT(*) FROM app_notifications WHERE user_id IN (SELECT id FROM cleanup_mock_users)) AS mock_notifications;

SELECT u.id, u.username, u.full_name, u.rank, u.source_employee_number, d.name AS district
FROM users u
LEFT JOIN districts d ON d.id = u.district_id
WHERE u.id IN (SELECT id FROM cleanup_mock_users)
ORDER BY u.username
LIMIT 250;

ROLLBACK;
