# trezord-go

[![Build Status](https://travis-ci.org/trezor/trezord-go.svg?branch=master)](https://travis-ci.org/trezor/trezord-go) [![gitter](https://badges.gitter.im/trezor/community.svg)](https://gitter.im/trezor/community)

TREZOR Communication Daemon aka TREZOR Bridge (written in Go)

```
go build
./trezord-go -h
```

## Guide to compiling packages

Prerequisites:

* `go get github.com/karalabe/xgo`
* `docker pull karalabe/xgo-latest`
* make sure `xgo` and `docker` are in `$PATH`
* `cd release && make all`; the installers are in `installers`

## Quick guide to cross-compiling

Prerequisites:

* `go get github.com/karalabe/xgo`
* `docker pull karalabe/xgo-latest`

Compiling for officially supported platforms:

* `$GOPATH/bin/xgo -targets=windows/amd64,windows/386,darwin/amd64,linux/amd64,linux/386 github.com/trezor/trezord-go`

## API documentation

`trezord` starts server on `localhost`, with port `21324`. You can use `https`, by using `https://localback.net:21324` which redirects to localhost. You can call this web address with standard AJAX calls from websites (see the note about whitelisting).

Server supports following API calls:

| url <br> method | parameters | result type | description |
|-------------|------------|-------------|-------------|
| `/` <br> GET | | {`version`:&nbsp;string,<br> `configured`:&nbsp;boolean,<br> `validUntil`:&nbsp;timestamp} | Returns current version of bridge and info about configuration.<br>See `/configure` for more info. |
| `/configure` <br> POST | request body: config, as hex string | {} | Before any advanced call, configuration file needs to be loaded to bridge.<br> Configuration file is signed by SatoshiLabs and the validity of the signature is limited.<br>Current config should be [in this repo](https://github.com/trezor/webwallet-data/blob/master/config_signed.bin), or [on AWS here](https://wallet.trezor.io/data/config_signed.bin). |
| `/enumerate` <br> GET | | Array&lt;{`path`:&nbsp;string, <br>`session`:&nbsp;string&nbsp;&#124;&nbsp;null}&gt; | Lists devices.<br>`path` uniquely defines device between more connected devices. It might or might not be unique over time; on some platform it changes, on others given USB port always returns the same path.<br>If `session` is null, nobody else is using the device; if it's string, it identifies who is using it. |
| `/listen` <br> POST | request body: previous, as JSON | like `enumerate` | Listen to changes and returns either on change or after 30 second timeout. Compares change from `previous` that is sent as a parameter. "Change" is both connecting/disconnecting and session change. |
| `/acquire/PATH/PREVIOUS` <br> POST | `PATH`: path of device<br>`PREVNOUS`: previous session (or string "null") | {`session`:&nbsp;string} | Acquires the device at `PATH`. By "acquiring" the device, you are claiming the device for yourself.<br>Before acquiring, checks that the current session is `PREVIOUS`.<br>If two applications call `acquire` on a newly connected device at the same time, only one of them succeed. |
| `/release/SESSION`<br>POST | `SESSION`: session to release | {} | Releases the device with the given session.<br>By "releasing" the device, you claim that you don't want to use the device anymore. |
| `/call/SESSION`<br>POST | `SESSION`: session to call<br><br>request body: JSON <br>{`type`: string, `message`: object}  | {`type`: string, `body`: object} | Calls the message and returns the response from TREZOR.<br>Messages are defined in [this protobuf file](https://github.com/trezor/trezor-common/blob/master/protob/messages.proto).<br>`type` in request is, for example, `GetFeatures`; `type` in response is, for example, `Features` |

