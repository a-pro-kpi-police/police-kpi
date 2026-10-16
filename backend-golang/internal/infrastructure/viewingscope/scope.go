// Package viewingscope carries the authenticated request's viewing boundary.
// An absent boundary leaves existing roles unchanged; an empty boundary denies all rows.
package viewingscope

import (
	"context"
	"strings"
)

type key struct{}

func WithDistricts(ctx context.Context, ids []int) context.Context {
	return context.WithValue(ctx, key{}, append([]int{}, ids...))
}

// Predicate accepts only a column name supplied by repository code, never user input.
func Predicate(ctx context.Context, column string, args *[]any) string {
	ids, limited := ctx.Value(key{}).([]int)
	if !limited {
		return "1=1"
	}
	if len(ids) == 0 {
		return "1=0"
	}
	for _, id := range ids {
		*args = append(*args, id)
	}
	return column + " IN (" + strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ")"
}
