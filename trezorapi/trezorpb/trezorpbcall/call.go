// Package trezorpbcall are helper procedures for marshaling, unmarshalling,
// and calling at the same time.
//
// See example in the main github.com/trezor/trezord-go/trezorapi/
package trezorpbcall

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/trezor/trezord-go/trezorapi"
	"github.com/trezor/trezord-go/trezorapi/trezorpb/marshal"
)

// Call is similar to api.Call, but in addition to that,
// it automatically marshals and unmarshal from protobuf,
// which is very error-prone by hand
func Call(
	ctx context.Context,
	a *trezorapi.API,
	pbMessage proto.Message,
	session string,
	debugLink bool,
) (proto.Message, error) {
	message, err := marshal.Marshal(pbMessage)
	if err != nil {
		return nil, err
	}

	backMessage, err := a.Call(ctx, message, session, debugLink)
	if err != nil {
		return nil, err
	}
	return marshal.Unmarshal(backMessage)
}

// Post is similar to api.Post, but in addition to that,
// it automatically marshals protobuf,
// which is very error-prone by hand
func Post(
	ctx context.Context,
	a *trezorapi.API,
	pbMessage proto.Message,
	session string,
	debugLink bool,
) error {
	message, err := marshal.Marshal(pbMessage)
	if err != nil {
		return err
	}

	return a.Post(ctx, message, session, debugLink)
}

// Read is similar to api.Read, but in addition to that,
// it automatically unmarshals protobuf,
// which is very error-prone by hand
func Read(
	ctx context.Context,
	a *trezorapi.API,
	session string,
	debugLink bool,
) (proto.Message, error) {
	backMessage, err := a.Read(ctx, session, debugLink)
	if err != nil {
		return nil, err
	}
	return marshal.Unmarshal(backMessage)
}
