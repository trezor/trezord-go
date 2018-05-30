# Changelog

All notable changes to this project will be documented in this file.

## [2.0.14] - unreleased

- Add more devcon and msinfo output on windows
- Separate detailed log download as a different URL request
- Big refactor of http.go to smaller packages

## [2.0.13] - 2018-04-12

- Allowing nousb mode (with only emulator turned on)
- Adding /post for writes without reads (debug link, only emulator so far)
- Installing WDI only on Windows 7
- Remove existing WDI drivers on install, preventing double driver install
- Rework HID to use timeout reads to prevent windows crashes

## [2.0.12] - 2018-04-03

- Add devcon and wdi-simple tools for Windows device drivers manipulation
- Fix deadlock
- Preventing panic when request is closed

## [2.0.11] - 2018-03-22

- Using libusb rc4, fixing windows bugs long-term
- Adding status page
- Fixing errors with half-read USB messages
- Refactoring for less complexity, fix golinter issues

## [2.0.10] - 2018-03-13

- Workaround for libusb bug in Windows 10 (fixes trezor/trezor-core#165)
- Fixes conflict with manually installed udev rules for T1 (Linux).

## [2.0.9] - 2018-03-05

- Fixes communication for very old T1 bootloaders

## [2.0.8] - 2018-03-01

- Stability fix for Linux

## [2.0.7] - 2018-02-23

- Use origin checks for all requests (fixes #16)

## [2.0.6] - 2018-02-17

- Allowing CORS for more ports on localhost (5xxx, 8xxx)

## [2.0.5] - 2018-02-17

- Added optional UDP (for emulators for both T1 and T2)
- WebUSB: Fixing stealing by detecting closed device before reading (otherwise read/write may hang)

## [2.0.4] - 2018-02-15

- WebUSB: Increase timeout to 5 seconds

## [2.0.3] - 2018-02-14

- WebUSB: don't abort on failed Reset Device or Set Configuration
- WebUSB: reset the device handle after opening
- Wire: add sanity checks

## [2.0.2] - 2018-02-14

- WebUSB: increase timeout to 500 milliseconds

## [2.0.1] - 2018-02-13

- WebUSB: explicitly open USB Configuration before claiming the interface
- Errors: nicer error messages
