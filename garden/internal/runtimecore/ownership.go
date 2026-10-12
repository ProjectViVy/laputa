package runtimecore

import (
	"errors"
	"fmt"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/gofrs/flock"
	"os"
	"path/filepath"
	"sort"
)

type ownership struct{ locks []*flock.Flock }

func acquireOwnership(paths ...string) (*ownership, error) {
	canonical := map[string]bool{}
	for _, path := range paths {
		p, err := registry.CanonicalPath(path)
		if err != nil {
			return nil, err
		}
		canonical[p] = true
	}
	ordered := make([]string, 0, len(canonical))
	for p := range canonical {
		ordered = append(ordered, p)
	}
	sort.Strings(ordered)
	own := &ownership{}
	for _, p := range ordered {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, errors.Join(err, own.Close())
		}
		lock := flock.New(p + ".garden-owner.lock")
		ok, err := lock.TryLock()
		if err != nil || !ok {
			return nil, errors.Join(fmt.Errorf("owner_conflict: resource already owned"), err, own.Close())
		}
		own.locks = append(own.locks, lock)
		if err = os.Chmod(p+".garden-owner.lock", 0600); err != nil {
			return nil, errors.Join(err, own.Close())
		}
	}
	return own, nil
}
func (o *ownership) Close() error {
	var errs []error
	for i := len(o.locks) - 1; i >= 0; i-- {
		errs = append(errs, o.locks[i].Unlock())
	}
	o.locks = nil
	return errors.Join(errs...)
}
