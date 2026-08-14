native:
	CGO_ENABLED=1 go build -buildvcs=false $(GOFLAGS)

# Native (host arch only) desktop binary
desktop-bin:
	CGO_ENABLED=1 go build -o trezor-bridge-desktop ./desktop

# Universal macOS binary: Intel (amd64) + Apple Silicon (arm64)
desktop-bin-universal:
	@echo "Building amd64…"
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		go build -o trezor-bridge-desktop-amd64 ./desktop
	@echo "Building arm64…"
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build -o trezor-bridge-desktop-arm64 ./desktop
	lipo -create -output trezor-bridge-desktop \
		trezor-bridge-desktop-amd64 trezor-bridge-desktop-arm64
	rm -f trezor-bridge-desktop-amd64 trezor-bridge-desktop-arm64
	file trezor-bridge-desktop

FYNE ?= $(shell go env GOPATH)/bin/fyne

# Build a double-clickable macOS .app (or Windows/Linux package via fyne).
# Requires: go install fyne.io/fyne/v2/cmd/fyne@latest
# (newer installs: go install fyne.io/tools/cmd/fyne@latest)
# Note: fyne package builds host arch only; prefer desktop-app-macos for universal.
desktop:
	@test -x "$(FYNE)" || { echo "Install fyne CLI: go install fyne.io/fyne/v2/cmd/fyne@latest"; exit 1; }
	cd desktop && CGO_ENABLED=1 "$(FYNE)" package -os darwin -name "Trezor Bridge" -icon Icon.png -appID io.trezor.bridge.desktop -release
	@echo "Built: desktop/Trezor Bridge.app"

# Universal double-clickable macOS .app (Intel + Apple Silicon)
desktop-app-macos: desktop-bin-universal
	rm -rf "Trezor Bridge.app"
	mkdir -p "Trezor Bridge.app/Contents/MacOS" "Trezor Bridge.app/Contents/Resources"
	cp trezor-bridge-desktop "Trezor Bridge.app/Contents/MacOS/Trezor Bridge"
	cp desktop/Icon.png "Trezor Bridge.app/Contents/Resources/Icon.png"
	cp desktop/AppIcon.icns "Trezor Bridge.app/Contents/Resources/AppIcon.icns"
	printf '%s\n' \
		'<?xml version="1.0" encoding="UTF-8"?>' \
		'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
		'<plist version="1.0"><dict>' \
		'<key>CFBundleExecutable</key><string>Trezor Bridge</string>' \
		'<key>CFBundleIdentifier</key><string>io.trezor.bridge.desktop</string>' \
		'<key>CFBundleName</key><string>Trezor Bridge</string>' \
		'<key>CFBundleIconFile</key><string>AppIcon</string>' \
		'<key>CFBundlePackageType</key><string>APPL</string>' \
		'<key>CFBundleVersion</key><string>2.0.34</string>' \
		'<key>CFBundleShortVersionString</key><string>2.0.34</string>' \
		'<key>LSMinimumSystemVersion</key><string>10.13</string>' \
		'<key>NSHighResolutionCapable</key><true/>' \
		'</dict></plist>' > "Trezor Bridge.app/Contents/Info.plist"
	@# Refresh Finder/Dock icon cache for this bundle
	touch "Trezor Bridge.app"
	@echo "Built universal app:"; file "Trezor Bridge.app/Contents/MacOS/Trezor Bridge"
	@echo "Built: Trezor Bridge.app (double-click to run on Intel and Apple Silicon)"

build-release:
	make -C release clean all

.PHONY: native desktop desktop-bin desktop-bin-universal desktop-app-macos build-release
