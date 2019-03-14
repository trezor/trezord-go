package trezorapi

import (
	"context"

	"github.com/trezor/trezord-go/internal/core"
	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

func (a *API) Listen(ctx context.Context, previousEntries []types.EnumerateEntry) ([]types.EnumerateEntry, error) {
	entries, err := a.c.Listen(ctx, previousEntries)
	return entries, err
}

func (a *API) Enumerate() ([]types.EnumerateEntry, error) {
	return a.c.Enumerate()
}

func (a *API) Acquire(path string, previousSession *string, debugLink bool) (string, error) {
	return a.c.Acquire(path, previousSession, debugLink)
}

func (a *API) Release(session string, debugLink bool) error {
	return a.c.Release(session, debugLink)
}

func (a *API) Call(
	ctx context.Context,
	message *types.Message,
	session string,
	debugLink bool,
) (*types.Message, error) {
	return a.c.Call(ctx, message, session, core.CallModeReadWrite, debugLink)
}

func (a *API) Post(
	ctx context.Context,
	message *types.Message,
	session string,
	debugLink bool,
) error {
	_, err := a.c.Call(ctx, message, session, core.CallModeWrite, debugLink)
	return err
}

func (a *API) Read(
	ctx context.Context,
	session string,
	debugLink bool,
) (*types.Message, error) {
	return a.c.Call(ctx, nil, session, core.CallModeRead, debugLink)
}
