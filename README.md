# onekey-bridge
可以在 [release](https://github.com/OneKeyHQ/onekey-bridge/releases) 看到最新发布的二进制安装文件。

## 编译与安装
在本地安装了 go 环境之后，最好把项目安装到 GOPATH 中，防止后续 go 编译错误，一般来说 GOPATH 是 `~/${username}/go` （`username` 是当前登陆账户）：
```
go get github.com/karalabe/xgo
docker pull karalabe/xgo-latest
```

同时确保 `xgo` 和 `docker` 在环境变量中

```
cd release
GOPATH=xx make all
```

进入 release 文件夹，在跑命令时，增加 GOPATH 环境变量，GOPATH 一般来说是 `~/${username}/go` （username 是当前登陆账户），也可以通过 `go env` 来确认。

编译完成之后，installers 下面就会有以下二进制安装文件：

* onekey-bridge-${version}-1.i386.rpm（Linux 32-bit (rpm)）
* onekey-bridge-${version}-1.x86_64.rpm（Linux 64-bit (rpm)）
* onekey-bridge-${version}-win32-install.exe（windows 系统用户，32位，64位均用此安装包）
* onekey-bridge-${version}.pkg（OSX 系统用户）
* onekey-bridge_${version}_amd64.deb（Linux 64-bit (deb)）
* onekey-bridge_${version}_i386.deb（Linux 32-bit (deb)）

其中中间的内容是版本号，由根目录下的 VERSION 文件控制，所以每次代码更新后，需要手动更新 VERSION 文件中的版本号。

**Only compatible with Chrome (version 53 or later) and Firefox (version 55 or later).**

status: [spec](https://w3c.github.io/webappsec-secure-contexts/#is-origin-trustworthy) [Chrome](https://bugs.chromium.org/p/chromium/issues/detail?id=607878) [Firefox](https://bugzilla.mozilla.org/show_bug.cgi?id=903966) [Edge](https://developer.microsoft.com/en-us/microsoft-edge/platform/issues/11963735/)

## Install and run from source

onekey-bridge requires go >= 1.6

```
go get github.com/OneKeyHQ/onekey-bridge
go build github.com/OneKeyHQ/onekey-bridge
./onekey-bridge -h
```

On Linux don't forget to install the [udev rules](https://github.com/trezor/trezor-common/blob/master/udev/51-trezor.rules) if you are running from source and not using pre-built packages.

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

* `$GOPATH/bin/xgo -targets=windows/amd64,windows/386,darwin/amd64,linux/amd64,linux/386 .`

## Emulator support

onekey supports emulators for both OneKey versions. However, you need to enable it manually; it is disabled by default. After enabling, services that work with emulator can work with all services that support onekey.

To enable emulator, run onekey with a parameter `-e` followed by port, for every emulator with an enabled port

`./onekey -e 21324`

If you want to run this automatically on linux, do

`sudo systemctl edit --full onekey.service`

and edit the service file (and maybe restart the onekey service). On mac, you will need to edit

`/Library/LaunchAgents/com.bitcoinonekey.onekeyBridge.onekey.plist`

and edit the last `<string>` in the plist. (And also probably restart the pc.)

You can disable all USB in order to run on some virtuaized environments, for example Travis

`./onekey -e 21324 -u=false`

## API documentation

`onekey-bridge` starts a HTTP server on `http://localhost:21320`. AJAX calls are only enabled from onekey.so subdomains.

Server supports following API calls:

| url <br> method | parameters | result type | description |
|-------------|------------|-------------|-------------|
| `/` <br> POST | | {`version`:&nbsp;string} | Returns current version of bridge |
| `/enumerate` <br> POST | | Array&lt;{`path`:&nbsp;string, <br>`session`:&nbsp;string&nbsp;&#124;&nbsp;null}&gt; | Lists devices.<br>`path` uniquely defines device between more connected devices. Two different devices (or device connected and disconnected) will return different paths.<br>If `session` is null, nobody else is using the device; if it's string, it identifies who is using it. |
| `/listen` <br> POST | request body: previous, as JSON | like `enumerate` | Listen to changes and returns either on change or after 30 second timeout. Compares change from `previous` that is sent as a parameter. "Change" is both connecting/disconnecting and session change. |
| `/acquire/PATH/PREVIOUS` <br> POST | `PATH`: path of device<br>`PREVIOUS`: previous session (or string "null") | {`session`:&nbsp;string} | Acquires the device at `PATH`. By "acquiring" the device, you are claiming the device for yourself.<br>Before acquiring, checks that the current session is `PREVIOUS`.<br>If two applications call `acquire` on a newly connected device at the same time, only one of them succeed. |
| `/release/SESSION`<br>POST | `SESSION`: session to release | {} | Releases the device with the given session.<br>By "releasing" the device, you claim that you don't want to use the device anymore. |
| `/call/SESSION`<br>POST | `SESSION`: session to call<br><br>request body: hexadecimal string | hexadecimal string | Both input and output are hexadecimal, encoded in following way:<br>first 2 bytes (4 characters in the hexadecimal) is the message type<br>next 4 bytes (8 in hex) is length of the data<br>the rest is the actual encoded protobuf data.<br>Protobuf messages are defined in [this protobuf file](https://github.com/trezor/trezor-common/blob/master/protob/messages.proto) and the app, calling onekey, should encode/decode it itself. |
| `/post/SESSION`<br>POST | `SESSION`: session to call<br><br>request body: hexadecimal string | 0 | Similar to `call`, just doesn't read response back. Usable mainly for debug link. |
| `/read/SESSION`<br>POST | `SESSION`: session to call | 0 | Similar to `call`, just doesn't post, only reads. Usable mainly for debug link. |

## Debug link support

onekey has support for debug link.

To support an emulator with debug link, run

`./onekey -ed 21324:21320 -u=false`

this will detect emulator debug link on port 21320, with regular device on 21324.

To support WebUSB devices with debug link, no option is needed, just run onekey-bridge.

In the `enumerate` and `listen` results, there are now two new fields: `debug` and `debugSession`. `debug` signals that device can receive debug link messages.

Session management is separate for debug link and normal interface, so you can have two applications - one controlling onekey and one "normal".

There are new calls:

* `/debug/acquire/PATH`, which has the same path as normal `acquire`, and returns a `SESSION`
* `/debug/release/SESSION` releases session
* `/debug/call/SESSION`, `/debug/post/SESSION`, `/debug/read/SESSION` work as with normal interface

The session IDs for debug link start with the string "debug".

## Copyright

* (C) 2018 Karel Bilek, Jan Pochyla
* CORS Copyright (c) 2013 The Gorilla Handlers Authors, [BSD license](https://github.com/gorilla/handlers/blob/master/LICENSE)
* (c) 2017 Jason T. Harris (also see https://github.com/deadsy/libusb for comprehensive list)
* (C) 2017 Péter Szilágyi (also see https://github.com/karalabe/hid for comprehensive list)
* (C) 2010-2016 Pete Batard <pete@akeo.ie> (also see https://github.com/pbatard/libwdi/ for comprehensive list)
* Licensed under LGPLv3
