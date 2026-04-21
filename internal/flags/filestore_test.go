package flags

import (
	"context"
	"errors"
	"testing"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	s, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return s
}

func TestFileStore_CreateGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	flag := FeatureFlag{Key: "new_checkout", Enabled: true, Description: "try it"}
	if err := s.Create(ctx, flag); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "new_checkout")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Key != flag.Key || got.Enabled != flag.Enabled || got.Description != flag.Description {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set: %+v", got)
	}
}

func TestFileStore_CreateDuplicate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	flag := FeatureFlag{Key: "dup", Enabled: false}
	if err := s.Create(ctx, flag); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Create(ctx, flag); !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists, got %v", err)
	}
}

func TestFileStore_GetMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestFileStore_UpdatePreservesCreatedAt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.Create(ctx, FeatureFlag{Key: "k", Enabled: false}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	original, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if err := s.Update(ctx, FeatureFlag{Key: "k", Enabled: true, Description: "on"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("CreatedAt changed: was %v, now %v", original.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(original.UpdatedAt) && !updated.UpdatedAt.Equal(original.UpdatedAt) {
		t.Fatalf("UpdatedAt not advanced: was %v, now %v", original.UpdatedAt, updated.UpdatedAt)
	}
	if !updated.Enabled || updated.Description != "on" {
		t.Fatalf("fields not updated: %+v", updated)
	}
}

func TestFileStore_UpdateMissing(t *testing.T) {
	s := newTestStore(t)
	err := s.Update(context.Background(), FeatureFlag{Key: "ghost", Enabled: true})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestFileStore_Delete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.Create(ctx, FeatureFlag{Key: "gone", Enabled: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(ctx, "gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
	if err := s.Delete(ctx, "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound on repeat delete, got %v", err)
	}
}

func TestFileStore_ListSorted(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for _, k := range []string{"charlie", "alpha", "bravo"} {
		if err := s.Create(ctx, FeatureFlag{Key: k}); err != nil {
			t.Fatalf("Create %s: %v", k, err)
		}
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := []string{list[0].Key, list[1].Key, list[2].Key}
	want := []string{"alpha", "bravo", "charlie"}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("List order: got %v, want %v", got, want)
		}
	}
}

func TestFileStore_InvalidKey(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for _, bad := range []string{"", "../escape", "has space", "slash/key"} {
		if err := s.Create(ctx, FeatureFlag{Key: bad}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Create(%q): want ErrInvalid, got %v", bad, err)
		}
	}
}
