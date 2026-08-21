-- Delete auto-seeded mock users and their KPI data from production PostgreSQL.
-- Run deploy/preview-prod-mock-data.sql first and make a pg_dump backup.

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

DELETE FROM direct_messages
WHERE sender_id IN (SELECT id FROM cleanup_mock_users)
   OR recipient_id IN (SELECT id FROM cleanup_mock_users)
   OR entry_id IN (SELECT id FROM cleanup_mock_entries);

DELETE FROM app_notifications
WHERE user_id IN (SELECT id FROM cleanup_mock_users);

DELETE FROM kpi_targets
WHERE officer_id IN (SELECT id FROM cleanup_mock_users);

DELETE FROM kpi_entries
WHERE id IN (SELECT id FROM cleanup_mock_entries);

DELETE FROM users
WHERE id IN (SELECT id FROM cleanup_mock_users);

DELETE FROM import_batches b
WHERE (b.source_name = 'monthly_kpi_2026_07.csv'
    OR b.source_sheet = 'Месячная_таблица-1')
  AND NOT EXISTS (
    SELECT 1
    FROM kpi_entries e
    WHERE e.import_batch_id = b.id
  );

COMMIT;
