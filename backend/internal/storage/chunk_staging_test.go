package storage

import (
	"strings"
	"testing"
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

func TestLocalChunkStagingRejectsTraversal(t *testing.T) {
	staging := NewLocalChunkStaging(t.TempDir())
	if _, _, err := staging.Write("../../outside", 0, strings.NewReader("data")); err == nil {
		t.Fatal("expected traversal upload id to be rejected")
	}
}
