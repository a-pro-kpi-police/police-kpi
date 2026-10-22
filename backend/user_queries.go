package main

import (
	"context"
	"database/sql"

	domainusers "kpi-police/backend/internal/domain/users"
	infrausers "kpi-police/backend/internal/infrastructure/users"
)

func (a *App) getUserByID(ctx context.Context, id int) (*User, error) {
	return queryUserByID(ctx, a.db, id)
}

func (a *App) getUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := a.userRepository().GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return userModel(user), nil
}

func (a *App) getOfficer(ctx context.Context, id int) (*User, error) {
	user, err := a.userRepository().GetOfficer(ctx, id)
	if err != nil {
		return nil, err
	}
	return userModel(user), nil
}

func (a *App) getDistrict(ctx context.Context, id int) (*District, error) {
	district, err := a.userRepository().GetDistrict(ctx, id)
	if err != nil {
		return nil, err
	}
	return districtModel(district), nil
}

func queryUserByID(ctx context.Context, q dbtx, id int) (*User, error) {
	user, err := infrausers.NewRepository(q).GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return userModel(user), nil
}

func districtModels(districts []domainusers.District) []District {
	result := make([]District, 0, len(districts))
	for _, district := range districts {
		result = append(result, *districtModel(district))
	}
	return result
}

func userModel(user domainusers.User) *User {
	result := &User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		FullName:     user.FullName,
		Rank:         user.Rank,
		Role:         user.Role,
		MonthlyNorm:  user.MonthlyNorm,
	}
	if user.DistrictID != nil {
		result.DistrictID = sql.NullInt64{Int64: int64(*user.DistrictID), Valid: true}
	}
	if user.SourceEmployeeNumber != nil {
		result.SourceEmployeeNumber = sql.NullString{String: *user.SourceEmployeeNumber, Valid: true}
	}
	if user.District != nil {
		result.District = districtModel(*user.District)
	}
	return result
}

func districtModel(district domainusers.District) *District {
	return &District{
		ID:    district.ID,
		Name:  district.Name,
		Coeff: district.Coeff,
	}
}

func userOut(user *User) UserOut {
	return UserOut{
		ID:          user.ID,
		Username:    user.Username,
		FullName:    user.FullName,
		Rank:        user.Rank,
		Role:        user.Role,
		MonthlyNorm: round2(user.MonthlyNorm),
		District:    user.District,
	}
}
