package persistence

import (
	"context"
	"sync"

	"gorm.io/gorm"
)

type transactionContextKey struct{}

type scheduleMutationContextKey struct{}

var scheduleMutationMutex sync.Mutex

// ScheduleMutationSerialized reports whether ctx already owns the process-wide
// schedule-mutation serialization boundary.
func ScheduleMutationSerialized(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	held, _ := ctx.Value(scheduleMutationContextKey{}).(bool)
	return held
}

// SerializeScheduleMutation provides one process-wide lock with context-based
// re-entrancy. Mutation repositories acquire it before opening their database
// transaction, and nested scheduler calls inherit the marker, preventing the
// mutex/database lock-order inversion that would otherwise deadlock.
func SerializeScheduleMutation(ctx context.Context) (context.Context, func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	if held, _ := ctx.Value(scheduleMutationContextKey{}).(bool); held {
		return ctx, func() {}
	}
	scheduleMutationMutex.Lock()
	lockedContext := context.WithValue(ctx, scheduleMutationContextKey{}, true)
	return lockedContext, scheduleMutationMutex.Unlock
}

func WithTransaction(ctx context.Context, transaction *gorm.DB) context.Context {
	if transaction == nil {
		return ctx
	}
	return context.WithValue(ctx, transactionContextKey{}, transaction)
}

func Transaction(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if ctx != nil {
		if transaction, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok && transaction != nil {
			return transaction.WithContext(ctx)
		}
	}
	return fallback.WithContext(ctx)
}

const scheduleMutationAdvisoryLockKey int64 = 76061

// LockScheduleMutation serialises every confirmed mutation that can alter the
// portfolio schedule before any project/task row locks are taken. PostgreSQL
// advisory transaction locks also coordinate multiple API instances. SQLite
// test repositories remain deterministic without relying on unsupported SQL.
func LockScheduleMutation(transaction *gorm.DB) error {
	if transaction == nil || transaction.Dialector.Name() != "postgres" {
		return nil
	}
	return transaction.Exec("SELECT pg_advisory_xact_lock(?)", scheduleMutationAdvisoryLockKey).Error
}
