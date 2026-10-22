package main

import (
	"context"
	"database/sql"

	"kpi-police/backend/internal/infrastructure/bootstrap"
)

const (
	monthlyImportMode = bootstrap.MonthlyImportMode
	demoSeedMode      = bootstrap.DemoSeedMode
)

func (a *App) seed() error {
	logDebugf("bootstrap seed: start")
	err := bootstrap.Seed(context.Background(), bootstrap.Config{
		BeginTx: func(ctx context.Context, opts *sql.TxOptions) (bootstrap.Tx, error) {
			return a.db.BeginTx(ctx, opts)
		},
		HashPassword: hashPassword,
		RecalculateOfficerMonth: func(ctx context.Context, tx bootstrap.Tx, officerID int, month string) error {
			_, err := recalculateOfficerMonth(ctx, tx, officerID, month)
			return err
		},
	})
	if err != nil {
		logErrorf("bootstrap seed: failed: %v", err)
		return err
	}
	logDebugf("bootstrap seed: complete")
	return nil
}
