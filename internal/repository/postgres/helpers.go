package postgres

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgTimeTZ extracts time.Time from a pgtype.Timestamptz value.
// Falls back to a zero value if the timestamp is not valid.
func pgTimeTZ(ts pgtype.Timestamptz, _ *time.Location) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}

// parseUUID parses a string into an uuid.UUID, returning a descriptive error.
func parseUUID(s string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return u, nil
}

// uuidPtrToNullable converts an optional string UUID pointer to pgtype.UUID.
func uuidPtrToNullable(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{}
	}
	u, err := uuid.Parse(*s)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: u, Valid: true}
}

// intPtrToNullable converts an optional *int to pgtype.Int4.
func intPtrToNullable(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

// float64PtrToNullable converts an optional *float64 to pgtype.Numeric.
func float64PtrToNullable(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(*v)
	return n
}

// numericToFloat64 safely converts pgtype.Numeric to float64.
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	if f.Valid {
		return f.Float64
	}
	return 0
}

// usernamePtrToNullable safely converts string pointer to pgtype.Text.
func usernamePtrToNullable(s *string) pgtype.Text {
	if s == nil || len(*s) == 0 || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
