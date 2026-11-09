package engineabi

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

func IsLibrary(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".dylib" || ext == ".so"
}

func Execute(ctx context.Context, path, id string, request []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	library, err := open(path)
	if err != nil {
		return nil, err
	}

	finished := make(chan struct{})
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		cancelWhenRequested(ctx, finished, func() bool { return library.cancel(id) })
	}()
	response, status, err := library.execute(request)
	close(finished)
	<-joined
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	if status != 0 {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(response, &failure) == nil && failure.Error != "" {
			return nil, fmt.Errorf("engine ABI status %d: %s", status, failure.Error)
		}
		return nil, fmt.Errorf("engine ABI status %d", status)
	}
	return response, nil
}

func Check(path string) error {
	_, err := open(path)
	return err
}

func cancelWhenRequested(ctx context.Context, finished <-chan struct{}, cancel func() bool) {
	select {
	case <-finished:
		return
	case <-ctx.Done():
	}
	retry := time.NewTicker(10 * time.Millisecond)
	defer retry.Stop()
	for {
		select {
		case <-finished:
			return
		default:
		}
		if cancel() {
			return
		}
		select {
		case <-finished:
			return
		case <-retry.C:
		}
	}
}
