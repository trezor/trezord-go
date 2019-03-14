package trezorapi

import (
	"context"

	"github.com/trezor/trezord-go/internal/core"
	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
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
		path string,
		prevSession *string,
		debugLink bool,
	) (string, error)
	Release(
		session string,
		debugLink bool,
	) error
	Call(
		ctx context.Context,
		message *types.Message,
		session string,
		mode core.CallMode,
		debugLink bool,
	) (*types.Message, error)
}
