package longmemeval

import (
	"sync"
)

// runWorkerPool fans out work(items[i]) across up to `workers` goroutines,
// preserving order in the returned slice. If any invocation returns an error,
// the first error is returned and remaining results are zero-valued.
func runWorkerPool[T, R any](items []T, workers int, work func(T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	errs := make([]error, len(items))

	type job struct {
		idx  int
		item T
	}

	jobs := make(chan job, len(items))
	for i, item := range items {
		jobs <- job{idx: i, item: item}
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < workers && w < len(items); w++ {
		wg.Go(func() {
			for j := range jobs {
				r, err := work(j.item)
				results[j.idx] = r
				errs[j.idx] = err
			}
		})
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

// runWorkerPoolWithResource is like runWorkerPool but each worker gets its own
// dedicated resource of type Res (e.g. a dedicated embedder instance). The
// number of workers equals len(resources).
func runWorkerPoolWithResource[T, R, Res any](
	items []T,
	resources []Res,
	work func(Res, T) (R, error),
) ([]R, error) {
	results := make([]R, len(items))
	errs := make([]error, len(items))

	type job struct {
		idx  int
		item T
	}

	jobs := make(chan job, len(items))
	for i, item := range items {
		jobs <- job{idx: i, item: item}
	}
	close(jobs)

	var wg sync.WaitGroup
	for _, res := range resources {
		wg.Go(func() {
			for j := range jobs {
				r, err := work(res, j.item)
				results[j.idx] = r
				errs[j.idx] = err
			}
		})
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}
