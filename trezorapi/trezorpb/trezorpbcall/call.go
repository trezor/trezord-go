package trezorpbcall

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/trezor/trezord-go/trezorapi"
	"github.com/trezor/trezord-go/trezorapi/trezorpb/marshal"
)

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
