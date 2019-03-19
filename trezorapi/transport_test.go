package trezorapi_test

import (
	"context"
	"fmt"

	"github.com/trezor/trezord-go/trezorapi"
	"github.com/trezor/trezord-go/trezorapi/trezorpb"
	"github.com/trezor/trezord-go/trezorapi/trezorpb/trezorpbcall"
)

func Example() {
	// creating an API object
	a, err := trezorapi.New(trezorapi.AddUDPPort(21324))
	if err != nil {
		panic(err)
	}

	// enumerating
	ds, err := a.Enumerate()
	if err != nil {
		panic(err)
	}
	d := ds[0] // panics when len < 1

	// acquiring
	debugLink := false
	session, err := a.Acquire(d.Path, d.Session, debugLink)
	if err != nil {
		panic(err)
	}

	// calling, automatically marshaling/demarshaling PB messages
	res, err := trezorpbcall.Call(
		context.Background(),
		a,
		&trezorpb.Initialize{},
		session,
		debugLink,
	)
	if err != nil {
		panic(err)
	}
	switch typed := res.(type) {
	case *trezorpb.Features:
		fmt.Printf("Device ID: %s", *typed.DeviceId)
	default:
		fmt.Println("Unknown type.")
	}

	// releasing
	err = a.Release(session, debugLink)
	if err != nil {
		panic(err)
	}
}
