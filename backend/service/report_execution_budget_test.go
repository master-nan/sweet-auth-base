package service

import (
	"backend/dto/request"
	myerrors "backend/internal/errors"
	"backend/internal/metadata"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestReportExecutionDeadlinesAreExplicitlyTiered(t *testing.T) {
	if !(reportDesignPreviewDeadline < reportRuntimeRunDeadline && reportRuntimeRunDeadline < reportRuntimeExportDeadline) {
		t.Fatalf("deadlines must be tiered: design=%s run=%s export=%s", reportDesignPreviewDeadline, reportRuntimeRunDeadline, reportRuntimeExportDeadline)
	}
	for name, budget := range map[string]time.Duration{
		"design": reportDesignPreviewDeadline,
		"run":    reportRuntimeRunDeadline,
		"export": reportRuntimeExportDeadline,
	} {
		t.Run(name, func(t *testing.T) {
			started := time.Now()
			ctx, cancel := withReportExecutionDeadline(context.Background(), budget)
			defer cancel()
			deadline, ok := ctx.Deadline()
			if !ok || deadline.Before(started.Add(budget-time.Second)) || deadline.After(started.Add(budget+time.Second)) {
				t.Fatalf("deadline=%v budget=%s", deadline, budget)
			}
		})
	}

	parent, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer parentCancel()
	parentDeadline, _ := parent.Deadline()
	ctx, cancel := withReportExecutionDeadline(parent, reportRuntimeExportDeadline)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if deadline.After(parentDeadline) {
		t.Fatalf("report deadline %v exceeded caller deadline %v", deadline, parentDeadline)
	}
}

func TestReportPreviewEntryPointsCarryActionDeadline(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "preview_deadline_entries", reportStatusDraft, queryConfig, layoutConfig)
	if _, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{}); err != nil {
		t.Fatalf("publish report: %v", err)
	}
	metadataSpy := &reportDeadlineMetadataSpy{RuntimeReader: env.svc.metadataRuntime}
	env.svc.metadataRuntime = metadataSpy

	cases := []struct {
		name   string
		budget time.Duration
		run    func() error
	}{
		{name: "design-preview", budget: reportDesignPreviewDeadline, run: func() error {
			_, err := env.svc.DesignPreview(env.ctx, report.Id, request.ReportPreviewReq{})
			return err
		}},
		{name: "legacy-preview", budget: reportDesignPreviewDeadline, run: func() error {
			_, err := env.svc.Preview(env.ctx, report.Id, request.ReportPreviewReq{})
			return err
		}},
		{name: "runtime-run", budget: reportRuntimeRunDeadline, run: func() error {
			_, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{})
			return err
		}},
		{name: "runtime-export", budget: reportRuntimeExportDeadline, run: func() error {
			_, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
			return err
		}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			observedBefore := len(metadataSpy.remainingBudgets)
			if err := item.run(); err != nil {
				t.Fatalf("execute entry point: %v", err)
			}
			if len(metadataSpy.remainingBudgets) <= observedBefore {
				t.Fatal("metadata read did not receive execution context")
			}
			remaining := metadataSpy.remainingBudgets[observedBefore]
			if remaining > item.budget || remaining < item.budget-time.Second {
				t.Fatalf("observed budget = %s, want approximately %s", remaining, item.budget)
			}
		})
	}
}

func TestReportSQLFieldInferenceEntryCarriesDesignDeadline(t *testing.T) {
	recorder := newReportJoinedQueryRecorder(0, nil)
	queryDB := newReportJoinedQueryDB(t, recorder)
	recorder.deadlines = nil
	svc := &ReportService{reportRepo: reportDefinitionRepoWithQueryDB{queryDB: queryDB}}
	if _, err := svc.InferSQLFields(newReportV1ATestEnv(t, reportV1AUser(true)).ctx, request.ReportSQLFieldsReq{
		SQL: "SELECT 1 AS value",
	}); err != nil {
		t.Fatalf("infer SQL fields: %v", err)
	}
	if len(recorder.deadlines) == 0 {
		t.Fatal("SQL field inference query did not receive a deadline")
	}
	remaining := time.Until(recorder.deadlines[0])
	if remaining > reportDesignPreviewDeadline || remaining < reportDesignPreviewDeadline-time.Second {
		t.Fatalf("SQL field inference budget = %s, want approximately %s", remaining, reportDesignPreviewDeadline)
	}
}

func TestReportStatementTimeoutUsesRemainingSharedBudget(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(2500*time.Millisecond))
	defer cancel()
	milliseconds, err := reportStatementTimeoutMilliseconds(ctx, now)
	if err != nil {
		t.Fatalf("statement timeout: %v", err)
	}
	if milliseconds != 2500 {
		t.Fatalf("statement timeout = %dms, want 2500ms", milliseconds)
	}
}

func TestPostgresReportTransactionSetsLocalTimeoutBeforeCountAndData(t *testing.T) {
	recorder := newReportJoinedQueryRecorder(1, nil)
	db := newReportJoinedQueryDB(t, recorder)
	recorder.queries = nil
	recorder.args = nil
	db.Config.Dialector = reportNamedDialector{Dialector: db.Config.Dialector, name: "postgres"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	wantDeadline, _ := ctx.Deadline()
	seenDeadlines := make([]time.Time, 0, 2)
	err := withReportExecutionTransaction(ctx, db, func(tx *gorm.DB) error {
		for _, query := range []string{"SELECT COUNT(1) FROM demo", "SELECT value FROM demo"} {
			deadline, ok := tx.Statement.Context.Deadline()
			if !ok {
				t.Fatal("transaction query context is missing deadline")
			}
			seenDeadlines = append(seenDeadlines, deadline)
			var value int64
			if err := tx.Raw(query).Scan(&value).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("report transaction: %v", err)
	}
	if len(seenDeadlines) != 2 || !seenDeadlines[0].Equal(wantDeadline) || !seenDeadlines[1].Equal(wantDeadline) {
		t.Fatalf("count/data did not share deadline: got=%v want=%v", seenDeadlines, wantDeadline)
	}
	if len(recorder.queries) < 3 || !strings.Contains(recorder.queries[0], "set_config") ||
		!strings.Contains(strings.ToLower(recorder.queries[1]), "count") ||
		!strings.Contains(strings.ToLower(recorder.queries[2]), "select value") {
		t.Fatalf("query order = %#v, want SET LOCAL timeout then count and data", recorder.queries)
	}
}

func TestNormalizeReportExecutionErrorProducesTimeoutKind(t *testing.T) {
	err := normalizeReportExecutionError(context.DeadlineExceeded)
	if !errors.Is(err, context.DeadlineExceeded) || myerrors.KindOf(err) != myerrors.KindTimeout {
		t.Fatalf("normalized error = %v kind=%s", err, myerrors.KindOf(err))
	}
}

type reportNamedDialector struct {
	gorm.Dialector
	name string
}

type reportDeadlineMetadataSpy struct {
	metadata.RuntimeReader
	remainingBudgets []time.Duration
}

func (s *reportDeadlineMetadataSpy) GetTable(ctx context.Context, tableCode string) (metadata.TableMetadata, error) {
	deadline, ok := ctx.Deadline()
	if ok {
		s.remainingBudgets = append(s.remainingBudgets, time.Until(deadline))
	}
	return s.RuntimeReader.GetTable(ctx, tableCode)
}

func (d reportNamedDialector) Name() string {
	return d.name
}
