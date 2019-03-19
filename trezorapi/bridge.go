package trezorapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/trezor/trezord-go/internal/core"
	"github.com/trezor/trezord-go/internal/message"
	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

type bridge struct {
	url     string
	Version string
}

func (b *bridge) post(
	ctx context.Context,
	url string,
	body io.Reader,
	decode func(r io.Reader) error,
) error {
	req, err := http.NewRequest("POST", b.url+url, body)
	if err != nil {
		return err
	}
	req.Header.Add("Origin", "https://golang.trezor.io")
	req = req.WithContext(ctx)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err := r.Body.Close()
		if err != nil {
			// ??
			fmt.Println(err)
		}
	}()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status code %d", r.StatusCode)
	}

	err = decode(r.Body)
	if err != nil {
		return err
	}
	return nil
}

func newBridge(url string) (*bridge, error) {
	b := &bridge{url: url}

	var version types.VersionInfo
	err := b.post(context.Background(), "/", nil, func(d io.Reader) error {
		return json.NewDecoder(d).Decode(&version)
	})

	if err != nil {
		return nil, err
	}

	if strings.Split(version.Version, ".")[0] != "2" {
		return nil, fmt.Errorf("old version of bridge %s", version.Version)
	}
	b.Version = version.Version
	return b, nil
}

func (b *bridge) Enumerate() ([]types.EnumerateEntry, error) {
	var entries []types.EnumerateEntry
	err := b.post(context.Background(), "/enumerate", nil, func(d io.Reader) error {
		return json.NewDecoder(d).Decode(&entries)
	})

	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		e.Type = types.TypeBridgeTransport
	}
	return entries, nil
}

func (b *bridge) Listen(ctx context.Context, entries []types.EnumerateEntry) ([]types.EnumerateEntry, error) {
	var bufEntries bytes.Buffer
	err := json.NewEncoder(&bufEntries).Encode(entries)
	if err != nil {
		return nil, err
	}

	var resEntries []types.EnumerateEntry

	// context cancels request with err as expected
	err = b.post(ctx, "/listen", &bufEntries, func(d io.Reader) error {
		return json.NewDecoder(d).Decode(&resEntries)
	})

	if err != nil {
		return nil, err
	}
	for _, e := range resEntries {
		e.Type = types.TypeBridgeTransport
	}
	return resEntries, nil
}

func (b *bridge) Acquire(
	path string,
	prev *string,
	debug bool,
) (string, error) {
	if prev == nil || *prev == "" {
		nullStr := "null"
		prev = &nullStr
	}
	url := fmt.Sprintf("/acquire/%s/%s", path, *prev)
	if debug {
		url = "/debug" + url
	}

	var session types.SessionInfo

	err := b.post(context.Background(), url, nil, func(d io.Reader) error {
		return json.NewDecoder(d).Decode(&session)
	})
	if err != nil {
		return "", err
	}
	return session.Session, nil
}

func (b *bridge) Release(
	session string,
	debug bool,
) error {
	url := fmt.Sprintf("/release/%s", session)
	if debug {
		url = "/debug" + url
	}
	err := b.post(context.Background(), url, nil, func(d io.Reader) error {
		return nil // just ignore input
	})
	return err
}

func (b *bridge) Call(
	ctx context.Context,
	msg *types.Message,
	session string,
	mode core.CallMode,
	debug bool,
) (*types.Message, error) {
	var rurl string
	switch mode {
	case core.CallModeRead:
		rurl = "read"
	case core.CallModeWrite:
		rurl = "post"
	case core.CallModeReadWrite:
		rurl = "call"
	default:
		return nil, fmt.Errorf("strange call mode %d", mode)
	}
	if session == "" {
		session = "null"
	}

	url := fmt.Sprintf("/%s/%s", rurl, session)
	if debug {
		url = "/debug" + url
	}

	var hexreader io.Reader

	hexreader, err := writeHex(mode, msg)
	if err != nil {
		return nil, err
	}

	var resmsg *types.Message

	err = b.post(ctx, url, hexreader, func(d io.Reader) error {
		resMsgPin, rErr := readHex(mode, d)
		if rErr != nil {
			return rErr
		}
		resmsg = resMsgPin
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resmsg, nil
}

func readHex(mode core.CallMode, d io.Reader) (*types.Message, error) {
	if mode != core.CallModeWrite {
		reshexbytes, err := ioutil.ReadAll(d)
		if err != nil {
			return nil, err
		}
		resbytes := make([]byte, hex.DecodedLen(len(reshexbytes)))
		_, err = hex.Decode(resbytes, reshexbytes)
		if err != nil {
			return nil, err
		}
		resmsg, err := message.FromBridgeFormat(resbytes, nil)
		if err != nil {
			return nil, err
		}
		return resmsg, nil
	}
	return nil, nil
}

func writeHex(mode core.CallMode, msg *types.Message) (io.Reader, error) {
	if mode != core.CallModeRead {
		body, err := message.ToBridgeFormat(msg, nil)
		if err != nil {
			return nil, err
		}
		hexbody := hex.EncodeToString(body)
		return strings.NewReader(hexbody), nil
	}
	return nil, nil
}

func (b *bridge) Close() {}
