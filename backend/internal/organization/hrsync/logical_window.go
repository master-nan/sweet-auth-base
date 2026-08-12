package hrsync

import (
	"errors"
	"time"
)

var ErrLogicalWindowInvalid = errors.New("org_sync_logical_window_invalid")

type WindowClassification string

const (
	WindowRecordReplay  WindowClassification = "lookback_replay"
	WindowRecordCurrent WindowClassification = "current"
	WindowRecordFuture  WindowClassification = "future"
)

// ClassifySourceChangeTime 只维护业务半开窗口；它不声称源响应具有上界。
func ClassifySourceChangeTime(sourceChangedAt, logicalStart, logicalEnd time.Time) (WindowClassification, error) {
	if !logicalEnd.After(logicalStart) {
		return "", ErrLogicalWindowInvalid
	}
	value, start, end := sourceChangedAt.UTC(), logicalStart.UTC(), logicalEnd.UTC()
	if value.Before(start) {
		return WindowRecordReplay, nil
	}
	if !value.Before(end) {
		return WindowRecordFuture, nil
	}
	return WindowRecordCurrent, nil
}
