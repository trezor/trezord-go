native:
	CGO_ENABLED=1 go build -buildvcs=false $(GOFLAGS)

desktop-bin:
	CGO_ENABLED=1 go build -o trezor-bridge-desktop ./desktop

FYNE ?= $(shell go env GOPATH)/bin/fyne

# Build a double-clickable macOS .app (or Windows/Linux package via fyne).
# Requires: go install fyne.io/fyne/v2/cmd/fyne@latest
# (newer installs: go install fyne.io/tools/cmd/fyne@latest)
desktop:
	@test -x "$(FYNE)" || { echo "Install fyne CLI: go install fyne.io/fyne/v2/cmd/fyne@latest"; exit 1; }
	cd desktop && CGO_ENABLED=1 "$(FYNE)" package -os darwin -name "Trezor Bridge" -icon Icon.png -appID io.trezor.bridge.desktop -release
	@echo "Built: desktop/Trezor Bridge.app"

# Fallback macOS .app bundle without fyne CLI
desktop-app-macos: desktop-bin
	rm -rf "Trezor Bridge.app"
	mkdir -p "Trezor Bridge.app/Contents/MacOS" "Trezor Bridge.app/Contents/Resources"
	cp trezor-bridge-desktop "Trezor Bridge.app/Contents/MacOS/Trezor Bridge"
	cp desktop/Icon.png "Trezor Bridge.app/Contents/Resources/Icon.png"
	printf '%s\n' \
		'<?xml version="1.0" encoding="UTF-8"?>' \
		'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
		'<plist version="1.0"><dict>' \
		'<key>CFBundleExecutable</key><string>Trezor Bridge</string>' \
		'<key>CFBundleIdentifier</key><string>io.trezor.bridge.desktop</string>' \
		'<key>CFBundleName</key><string>Trezor Bridge</string>' \
		'<key>CFBundlePackageType</key><string>APPL</string>' \
		'<key>CFBundleVersion</key><string>2.0.34</string>' \
		'<key>CFBundleShortVersionString</key><string>2.0.34</string>' \
		'<key>LSMinimumSystemVersion</key><string>10.13</string>' \
		'<key>NSHighResolutionCapable</key><true/>' \
		'</dict></plist>' > "Trezor Bridge.app/Contents/Info.plist"
	@echo "Built: Trezor Bridge.app (double-click to run)"

build-release:
	make -C release clean all

.PHONY: native desktop desktop-bin desktop-app-macos build-release
