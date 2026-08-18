# Monthly KPI import snapshots

`monthly_kpi_2026_07.csv` is a wide, auditable snapshot extracted from the
`Месячная_таблица-1` table in the source Numbers workbook.

- Blank KPI cells are stored as numeric zeroes.
- Missing employee coefficients use the neutral coefficient `1`.
- `scripts/export_monthly_numbers.js` regenerates the snapshot and the active
  KPI catalog from the workbook without modifying the source file.
