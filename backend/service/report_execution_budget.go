package service

import (
	myerrors "backend/internal/errors"
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const (
	reportDesignPreviewDeadline = 10 * time.Second
	reportRuntimeRunDeadline    = 30 * time.Second
	reportRuntimeExportDeadline = 2 * time.Minute
	reportExecutionContextKey   = "report_execution_context"
)

type reportSQLStateError interface {
	SQLState() string
}

func withReportExecutionDeadline(ctx context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, budget)
}

func withReportExecutionTransaction(ctx context.Context, db *gorm.DB, execute func(*gorm.DB) error) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := setPostgresReportStatementTimeout(ctx, tx); err != nil {
			return err
		}
		return execute(tx)
	})
}

func setPostgresReportStatementTimeout(ctx context.Context, tx *gorm.DB) error {
	if tx == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	milliseconds, err := reportStatementTimeoutMilliseconds(ctx, time.Now())
	if err != nil {
		return err
	}
	return applyPostgresReportStatementTimeout(tx, milliseconds)
}

func applyPostgresReportStatementTimeout(tx *gorm.DB, milliseconds int64) error {
	var applied string
	return tx.Raw(
		"SELECT set_config('statement_timeout', ?, true)",
		strconv.FormatInt(milliseconds, 10),
	).Scan(&applied).Error
}

func reportStatementTimeoutMilliseconds(ctx context.Context, now time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0, myerrors.WrapApplicationError(
			context.DeadlineExceeded,
			myerrors.KindTimeout,
			myerrors.CategoryBusiness,
			myerrors.ErrorCodeGeneric,
			"报表执行缺少截止时间",
		)
	}
	remaining := deadline.Sub(now)
	if remaining <= 0 {
		return 0, context.DeadlineExceeded
	}
	milliseconds := remaining.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	return milliseconds, nil
}

func normalizeReportExecutionError(err error) error {
	if err == nil {
		return nil
	}
	timedOut := errors.Is(err, context.DeadlineExceeded)
	var sqlStateErr reportSQLStateError
	if errors.As(err, &sqlStateErr) && sqlStateErr.SQLState() == "57014" {
		timedOut = true
	}
	if !timedOut {
		return err
	}
	return myerrors.WrapApplicationError(
		err,
		myerrors.KindTimeout,
		myerrors.CategoryBusiness,
		myerrors.ErrorCodeGeneric,
		"报表执行超时",
	)
}
