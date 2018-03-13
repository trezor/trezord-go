# Changelog
All notable changes to this project will be documented in this file.


## [2.0.10] - 2018-03-13
- Workaround for libusb bug in Windows 10 (fixes #28)
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
