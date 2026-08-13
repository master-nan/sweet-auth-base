package hrsync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNormalizeResignedEmployeeSourceUsesDateAndExplicitSourceClock(t *testing.T) {
	location := time.FixedZone("source", 8*60*60)
	input, err := (Normalizer{SourceSystemCode: OrganizationHRSourceSystemCode, SourceLocation: location}).NormalizeResignedEmployeeSource(HRResignedEmployeeSourceDTO{
		SourceID: "employee-1", ChangeTime: "2026-08-12T10:30:00", ResignedAt: "2026-08-10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if input.Key.ObjectKind() != ObjectKindEmployee || !input.SourceChangedAt.Equal(time.Date(2026, 8, 12, 2, 30, 0, 0, time.UTC)) ||
		!input.ResignedOn.Equal(time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("input=%+v", input)
	}
	for _, resignedAt := range []string{"", "2026-02-30", "1889-01-01", "2026-08-10T12:00:00"} {
		_, err := (Normalizer{SourceSystemCode: OrganizationHRSourceSystemCode, SourceLocation: location}).NormalizeResignedEmployeeSource(HRResignedEmployeeSourceDTO{
			SourceID: "employee-1", ChangeTime: "2026-08-12T10:30:00", ResignedAt: resignedAt,
		})
		if err == nil {
			t.Fatalf("resignedAt=%q accepted", resignedAt)
		}
	}
}

func TestAssignmentSourceParserSafeSubset(t *testing.T) {
	parser, err := NewAssignmentSourceParser(time.UTC, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raw := `[{"ID":"relation-1","公司ID":"company-ncid","兼职架构":1,"兼职类型":"part_time","部门主键ID":"department-ncid","岗位ID":null,"开始时间":"2026-01-01T09:00","结束时间":"","在岗":"Y","结束兼职":"N","ignored":"value"}]`
	candidates, err := parser.Parse(raw)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("candidates=%+v err=%v", candidates, err)
	}
	if candidates[0].RelationID != "relation-1" || candidates[0].PositionNCID != "" || candidates[0].Period.State != AssignmentPeriodCurrent || candidates[0].Period.ValidTo != nil {
		t.Fatalf("candidate=%+v", candidates[0])
	}

	tests := []struct {
		name string
		raw  string
		err  error
	}{
		{"missing relation", `[{"公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","在岗":"Y","结束兼职":"N"}]`, ErrAssignmentSourceInvalid},
		{"duplicate relation", `[{"ID":"r","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","在岗":"Y","结束兼职":"N"},{"ID":"r","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","在岗":"Y","结束兼职":"N"}]`, ErrAssignmentSourceConflict},
		{"historical empty end", `[{"ID":"r","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","结束时间":"","在岗":"N","结束兼职":"Y"}]`, ErrAssignmentPeriodInvalid},
		{"reverse period", `[{"ID":"r","公司ID":"c","部门主键ID":"d","开始时间":"2026-02-01","结束时间":"2026-01-01","在岗":"N","结束兼职":"Y"}]`, ErrAssignmentPeriodInvalid},
		{"status conflict", `[{"ID":"r","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","结束时间":"2026-02-01","在岗":"Y","结束兼职":"Y"}]`, ErrAssignmentStatusConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parser.Parse(test.raw)
			if !errors.Is(err, test.err) {
				t.Fatalf("err=%v want=%v", err, test.err)
			}
		})
	}
}

func TestAssignmentSourceParserLimitsAndCrosswalkGate(t *testing.T) {
	parser, err := NewAssignmentSourceParser(time.UTC, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	values := make([]string, MaxAssignmentSourceItems)
	for index := range values {
		values[index] = fmt.Sprintf(`{"ID":"r-%d","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","在岗":"Y","结束兼职":"N"}`, index)
	}
	if _, err := parser.Parse("[" + strings.Join(values, ",") + "]"); err != nil {
		t.Fatalf("maximum assignment count rejected: %v", err)
	}
	values = append(values, `{"ID":"overflow","公司ID":"c","部门主键ID":"d","开始时间":"2026-01-01","在岗":"Y","结束兼职":"N"}`)
	if _, err := parser.Parse("[" + strings.Join(values, ",") + "]"); !errors.Is(err, ErrAssignmentSourceInvalid) {
		t.Fatalf("count overflow err=%v", err)
	}
	if _, err := parser.Parse(strings.Repeat(" ", MaxAssignmentSourceBytes+1)); !errors.Is(err, ErrAssignmentSourceInvalid) {
		t.Fatalf("size overflow err=%v", err)
	}
	if _, err := (UnavailableOrganizationSourceCrosswalkResolver{}).Resolve(context.Background(), OrganizationSourceReference{
		SourceSystemCode: OrganizationHRSourceSystemCode, ObjectKind: ObjectKindManagementUnit, NCID: "ncid",
	}); !errors.Is(err, ErrSourceCrosswalkUnavailable) {
		t.Fatalf("crosswalk err=%v", err)
	}
}

func TestAssignmentSourceParserHandlesSanitizedShapeSet(t *testing.T) {
	parser, err := NewAssignmentSourceParser(time.UTC, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	values := make([]string, 0, 25)
	for index := 0; index < 25; index++ {
		position := fmt.Sprintf(`"岗位ID":"position-%d",`, index)
		if index%5 == 0 {
			position = `"岗位ID":null,`
		}
		onPost, ended, endedAt := "Y", "N", ""
		if index >= 13 {
			onPost, ended, endedAt = "N", "Y", "2026-06-30T18:00"
		}
		if index == 24 {
			endedAt = ""
		}
		values = append(values, fmt.Sprintf(`{"ID":"shape-%d","公司ID":"company-%d","兼职架构":%d,"兼职类型":"type-%d","部门主键ID":"department-%d",%s"开始时间":"2026-01-01T09:00","结束时间":"%s","在岗":"%s","结束兼职":"%s","variant_%d":true}`,
			index, index, index%2, index%3, index, position, endedAt, onPost, ended, index))
	}
	raw := "[" + strings.Join(values, ",") + "]"
	sources, err := ParseAssignmentSourceDTOs(raw)
	if err != nil || len(sources) != 25 {
		t.Fatalf("sanitized source shapes=%d err=%v", len(sources), err)
	}
	for index, source := range sources {
		period, periodErr := NormalizeAssignmentPeriod(source, time.UTC, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
		if index == 24 {
			if !errors.Is(periodErr, ErrAssignmentPeriodInvalid) {
				t.Fatalf("historical empty end err=%v", periodErr)
			}
			continue
		}
		if periodErr != nil || (index < 13 && period.State != AssignmentPeriodCurrent) || (index >= 13 && period.State != AssignmentPeriodHistorical) {
			t.Fatalf("period[%d]=%+v err=%v", index, period, periodErr)
		}
	}
	if _, err := parser.Parse(raw); !errors.Is(err, ErrAssignmentPeriodInvalid) {
		t.Fatalf("candidate normalization must reject historical empty end: %v", err)
	}
}
