package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalChunkStagingWriteMergeAndCleanup(t *testing.T) {
	staging := NewLocalChunkStaging(t.TempDir())
	first, _, err := staging.Write("upload-1", 0, strings.NewReader("abc"))
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := staging.Write("upload-1", 1, strings.NewReader("def"))
	if err != nil {
		t.Fatal(err)
	}
	merged, err := staging.Merge("upload-1", "result.txt", []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Size != 6 || merged.MD5 != "e80b5017098950fc58aad83c8c14978e" {
		t.Fatalf("unexpected merge result: %+v", merged)
	}
	reader, err := staging.Open(merged.Path)
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	if err = staging.Cleanup("upload-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = staging.Open(merged.Path); err == nil {
		t.Fatal("expected cleaned merged file to be absent")
	}
}

func TestLocalChunkStagingCleanupExpiredKeepsActiveAndPersistentFiles(t *testing.T) {
	base := t.TempDir()
	staging := NewLocalChunkStaging(base)
	oldPath, _, err := staging.Write("expired", 0, strings.NewReader("old"))
	if err != nil {
		t.Fatal(err)
	}
	activePath, _, err := staging.Write("active", 0, strings.NewReader("active"))
	if err != nil {
		t.Fatal(err)
	}
	persistent := filepath.Join(base, "2026", "08", "21", "saved.txt")
	if err := os.MkdirAll(filepath.Dir(persistent), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(persistent, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-48 * time.Hour)
	oldFull, _ := staging.safeFullPath(oldPath)
	if err := os.Chtimes(oldFull, old, old); err != nil {
		t.Fatal(err)
	}
	expiredDir := filepath.Dir(oldFull)
	if err := os.Chtimes(expiredDir, old, old); err != nil {
		t.Fatal(err)
	}

	removed, err := staging.CleanupExpired(time.Now(), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d, want 1", removed)
	}
	if _, err := staging.Open(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected expired staging file to be removed, got %v", err)
	}
	if reader, err := staging.Open(activePath); err != nil {
		t.Fatalf("active staging file was removed: %v", err)
	} else {
		_ = reader.Close()
	}
	if _, err := os.Stat(persistent); err != nil {
		t.Fatalf("persistent upload file was affected: %v", err)
	}
}

func TestLocalChunkStagingCleanupExpiredSkipsSymlinkSessions(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	staging := NewLocalChunkStaging(base)
	root, err := staging.safeFullPath("chunks")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	removed, err := staging.CleanupExpired(time.Now(), time.Hour)
	if err != nil || removed != 0 {
		t.Fatalf("symlink cleanup removed=%d err=%v", removed, err)
	}
}

func TestLocalChunkStagingRejectsTraversal(t *testing.T) {
	staging := NewLocalChunkStaging(t.TempDir())
	if _, _, err := staging.Write("../../outside", 0, strings.NewReader("data")); err == nil {
		t.Fatal("expected traversal upload id to be rejected")
	}
}
