package main

import (
	"context"
	"database/sql"
	"time"

	appdashboard "kpi-police/backend/internal/application/dashboard"
	appentries "kpi-police/backend/internal/application/entries"
	appmessenger "kpi-police/backend/internal/application/messenger"
	appnotifications "kpi-police/backend/internal/application/notifications"
	appscoring "kpi-police/backend/internal/application/scoring"
	apptargets "kpi-police/backend/internal/application/targets"
	appusers "kpi-police/backend/internal/application/users"
	domaindashboard "kpi-police/backend/internal/domain/dashboard"
	domainentries "kpi-police/backend/internal/domain/entries"
	domainmessenger "kpi-police/backend/internal/domain/messenger"
	domainnotifications "kpi-police/backend/internal/domain/notifications"
	domainscoring "kpi-police/backend/internal/domain/scoring"
	domaintargets "kpi-police/backend/internal/domain/targets"
	domainusers "kpi-police/backend/internal/domain/users"
	infracatalog "kpi-police/backend/internal/infrastructure/catalog"
	infradashboard "kpi-police/backend/internal/infrastructure/dashboard"
	infraentries "kpi-police/backend/internal/infrastructure/entries"
	inframessenger "kpi-police/backend/internal/infrastructure/messenger"
	infranotifications "kpi-police/backend/internal/infrastructure/notifications"
	infrascoring "kpi-police/backend/internal/infrastructure/scoring"
	infratargets "kpi-police/backend/internal/infrastructure/targets"
	infrausers "kpi-police/backend/internal/infrastructure/users"
)

func (a *App) catalogRepository() infracatalog.Repository {
	return infracatalog.NewRepository(a.db)
}

func (a *App) userRepository() infrausers.Repository {
	return infrausers.NewRepository(a.db)
}

func (a *App) userQueryService() appusers.QueryService {
	return appusers.NewQueryService(a.userRepository(), domainusers.NewEfficiencyPolicy())
}

func (a *App) dashboardService() appdashboard.Service {
	return appdashboard.NewService(
		infradashboard.NewRepository(a.db),
		a.userRepository(),
		domaindashboard.NewPolicy(),
	)
}

func (a *App) entryRepository() infraentries.Repository {
	return infraentries.NewRepository(a.db,
		func(ctx context.Context, opts *sql.TxOptions) (infraentries.Tx, error) {
			return a.db.BeginTx(ctx, opts)
		},
		func(ctx context.Context, tx infraentries.Tx, officerID int, month string) error {
			_, err := recalculateOfficerMonth(ctx, tx, officerID, month)
			return err
		},
	)
}

func (a *App) entryQueryService() appentries.QueryService {
	return appentries.NewQueryService(a.entryRepository(), domainentries.NewQueryPolicy())
}

func (a *App) messengerService() appmessenger.Service {
	return appmessenger.NewService(
		a.userRepository(),
		inframessenger.NewRepository(a.db),
		domainmessenger.NewPolicy(),
		func() time.Time { return time.Now().UTC() },
	)
}

func (a *App) notificationService() appnotifications.Service {
	return appnotifications.NewService(infranotifications.NewRepository(a.db), domainnotifications.NewPolicy())
}

func (a *App) targetRepository() infratargets.Repository {
	return infratargets.NewRepository(a.db, func(ctx context.Context, opts *sql.TxOptions) (infratargets.Tx, error) {
		return a.db.BeginTx(ctx, opts)
	})
}

func (a *App) targetQueryService() apptargets.QueryService {
	return apptargets.NewQueryService(a.targetRepository(), domaintargets.NewPolicy())
}

func scoringService(q dbtx) appscoring.Service {
	return appscoring.NewService(infrascoring.NewEntryRepository(q), domainscoring.NewPolicy())
}
