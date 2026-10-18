// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows

package renameio

import (
	"os"
	"path/filepath"
)

// PendingFile implements only the upstream API required by coder/hnsw. On
// Windows this implementation does not provide upstream's atomicity guarantee.
type PendingFile struct {
	*os.File
	path   string
	closed bool
	done   bool
}

// Cleanup removes the pending file after a failed replacement.
func (t *PendingFile) Cleanup() error {
	if t.done {
		return nil
	}
	var closeErr error
	if !t.closed {
		closeErr = t.Close()
		t.closed = true
	}
	if err := os.Remove(t.Name()); err != nil {
		return err
	}
	return closeErr
}

// CloseAtomicallyReplace provides the best-effort replacement operation used
// by coder/hnsw. os.Rename may not provide atomic replacement on Windows.
func (t *PendingFile) CloseAtomicallyReplace() error {
	if err := t.Sync(); err != nil {
		return err
	}
	if err := t.Close(); err != nil {
		t.closed = true
		return err
	}
	t.closed = true
	if err := os.Rename(t.Name(), t.path); err != nil {
		return err
	}
	t.done = true
	return nil
}

// TempFile creates a temporary file beside path by default. This keeps it on
// the destination volume so os.Rename can replace the derived index file.
func TempFile(dir, path string) (*PendingFile, error) {
	if dir == "" {
		dir = filepath.Dir(path)
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path))
	if err != nil {
		return nil, err
	}
	return &PendingFile{File: f, path: path}, nil
}
