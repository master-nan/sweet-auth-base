package service

import (
	"backend/enum"
	platformmetadata "backend/internal/metadata"
	"backend/model"
	"testing"
)

func TestApplyMenuDetailOpenModesUsesTableMetadata(t *testing.T) {
	menus := []model.SysMenu{
		{Name: "employee", TableCode: "org_employee"},
		{Name: "batch", TableCode: "org_sync_batch"},
		{Name: "directory"},
	}
	tables := []platformmetadata.TableMetadata{
		{Code: "org_employee", DetailOpenMode: enum.DetailOpenDialog},
		{Code: "org_sync_batch", DetailOpenMode: enum.DetailOpenPage},
	}

	applyMenuDetailOpenModes(menus, tables, nil)

	if menus[0].DetailOpenMode != enum.DetailOpenDialog {
		t.Fatalf("employee detail mode = %q, want dialog", menus[0].DetailOpenMode)
	}
	if menus[1].DetailOpenMode != enum.DetailOpenPage {
		t.Fatalf("batch detail mode = %q, want page", menus[1].DetailOpenMode)
	}
	if menus[2].DetailOpenMode != "" {
		t.Fatalf("directory detail mode = %q, want empty", menus[2].DetailOpenMode)
	}
}

func TestApplyMenuDetailOpenModesNormalizesInvalidMetadata(t *testing.T) {
	menus := []model.SysMenu{{Name: "employee", TableCode: "org_employee"}}
	tables := []platformmetadata.TableMetadata{{
		Code:           "org_employee",
		DetailOpenMode: enum.SysDetailOpenMode("invalid"),
	}}

	applyMenuDetailOpenModes(menus, tables, nil)

	if menus[0].DetailOpenMode != enum.DetailOpenAuto {
		t.Fatalf("detail mode = %q, want auto", menus[0].DetailOpenMode)
	}
}
