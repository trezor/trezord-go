package trezorapi

import (
	"context"

	"github.com/trezor/trezord-go/internal/core"
	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

// Listen waits for change in connection.
//
// The function stops execution until either some device is connected/disconnected,
// or it times out after a while on its own. So if this returns something, it is NOT
// guaranteed that there was change, and you should compare result to previousEntries.
//
// (This is not very go-like design, but it's like that for compatibility reasons.)
//
// If a device has different Path, it is guaranteed that it is a different connection
// (Paths are always unique per connection)
//
// Note - this is NOT true if user is using an older bridge (<2.0.25),
// where Path was instead a USB port identifier.
//
// Note - what also registers as change is change of session.
func (a *API) Listen(ctx context.Context, previousEntries []types.EnumerateEntry) ([]types.EnumerateEntry, error) {
	entries, err := a.c.Listen(ctx, previousEntries)
	return entries, err
}

// Enumerate returns all connected devices.
//
// Path is unique per "connection" - that is, physically disconnected and reconnected
// device has a different Path. However, when acquiring and releasing, Path stays the same.
func (a *API) Enumerate() ([]types.EnumerateEntry, error) {
	return a.c.Enumerate()
}

// Acquire takes the device and returns a session ID, which can be later used
// for calling.
//
// If bridge is used as a backend, this prevents multiple apps for grabing
// device simultaneously.
func (a *API) Acquire(path string, previousSession *string, debugLink bool) (string, error) {
	return a.c.Acquire(path, previousSession, debugLink)
}

// Release frees the device, other users can now use it.
func (a *API) Release(session string, debugLink bool) error {
	return a.c.Release(session, debugLink)
}

// Call is calling device with a given session.
//
// Note that this returns and requires Message object, which still is raw
// protobuf bytes. See github.com/trezor/trezord-go/trezorapi/trezorpb/trezorpbcall
// for how to call protobuf directly, or see example above.
func (a *API) Call(
	ctx context.Context,
	message *types.Message,
	session string,
	debugLink bool,
) (*types.Message, error) {
	return a.c.Call(ctx, message, session, core.CallModeReadWrite, debugLink)
}

// Post is calling device with a given session, but does not read back
//
// Note that this requires Message object, which still is raw
// protobuf bytes. See github.com/trezor/trezord-go/trezorapi/trezorpb/trezorpbcall
// for how to call protobuf directly, or see example above.
func (a *API) Post(
	ctx context.Context,
	message *types.Message,
	session string,
	debugLink bool,
) error {
	_, err := a.c.Call(ctx, message, session, core.CallModeWrite, debugLink)
	return err
}

// Read is reading 1 message from Trezor.
//
// Note that this returns Message object, which still is raw
// protobuf bytes. See github.com/trezor/trezord-go/trezorapi/trezorpb/trezorpbcall
// for how to call protobuf directly, or see example above.
func (a *API) Read(
	ctx context.Context,
	session string,
	debugLink bool,
) (*types.Message, error) {
	return a.c.Call(ctx, nil, session, core.CallModeRead, debugLink)
}
