package flags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

// keyPattern restricts flag keys to filesystem-safe characters to prevent
// path traversal via the key reaching os.Open/os.WriteFile.
var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

// FileStore persists each flag as a JSON file under a directory.
type FileStore struct {
	dir string
	mu  sync.RWMutex
	now func() time.Time
}

func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &FileStore{dir: dir, now: time.Now}, nil
}

func (s *FileStore) path(key string) string {
	return filepath.Join(s.dir, key+".json")
}

func validateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: key must match %s", ErrInvalid, keyPattern)
	}
	return nil
}

func (s *FileStore) Create(ctx context.Context, flag FeatureFlag) error {
	if err := validateKey(flag.Key); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.path(flag.Key)
	if _, err := os.Stat(path); err == nil {
		return ErrExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat flag: %w", err)
	}

	now := s.now().UTC()
	flag.CreatedAt = now
	flag.UpdatedAt = now
	return s.writeFile(path, flag)
}

func (s *FileStore) Get(ctx context.Context, key string) (FeatureFlag, error) {
	if err := validateKey(key); err != nil {
		return FeatureFlag{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readFile(s.path(key))
}

func (s *FileStore) List(ctx context.Context) ([]FeatureFlag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read storage dir: %w", err)
	}

	flags := make([]FeatureFlag, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		flag, err := s.readFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		flags = append(flags, flag)
	}

	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	return flags, nil
}

func (s *FileStore) Update(ctx context.Context, flag FeatureFlag) error {
	if err := validateKey(flag.Key); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.path(flag.Key)
	existing, err := s.readFile(path)
	if err != nil {
		return err
	}

	flag.CreatedAt = existing.CreatedAt
	flag.UpdatedAt = s.now().UTC()
	return s.writeFile(path, flag)
}

func (s *FileStore) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	return nil
}

func (s *FileStore) readFile(path string) (FeatureFlag, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return FeatureFlag{}, ErrNotFound
	}
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("read flag: %w", err)
	}

	var flag FeatureFlag
	if err := json.Unmarshal(data, &flag); err != nil {
		return FeatureFlag{}, fmt.Errorf("decode flag: %w", err)
	}
	return flag, nil
}

// writeFile does an atomic write via temp file + rename so a crash mid-write
// can't leave a partially-written JSON file behind.
func (s *FileStore) writeFile(path string, flag FeatureFlag) error {
	data, err := json.MarshalIndent(flag, "", "  ")
	if err != nil {
		return fmt.Errorf("encode flag: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, ".flag-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
