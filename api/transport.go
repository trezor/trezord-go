package api

import (
	"context"

	"github.com/trezor/trezord-go/internal/core"
	"github.com/trezor/trezord-go/types"
)

type transport interface {
	Enumerate() (
		[]types.EnumerateEntry,
		error,
	)
	Listen(
		ctx context.Context,
		entries []types.EnumerateEntry,
	) ([]types.EnumerateEntry, error)
	Acquire(
		path, prev string,
		debug bool,
	) (string, error)
	Release(
		session string,
		debug bool,
	) error
	Call(
		ctx context.Context,
		body []byte,
		session string,
		mode core.CallMode,
		debug bool,
	) ([]byte, error)
}
