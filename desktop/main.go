package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trezor/trezord-go/bridge"
	"github.com/trezor/trezord-go/core"
)

//go:embed Icon.png
var iconPNG []byte

func main() {
	a := app.NewWithID("io.trezor.bridge.desktop")
	a.Settings().SetTheme(&bridgeTheme{})
	a.SetIcon(fyne.NewStaticResource("Icon.png", iconPNG))
	w := a.NewWindow("Trezor Bridge")
	w.SetIcon(fyne.NewStaticResource("Icon.png", iconPNG))
	w.Resize(fyne.NewSize(420, 360))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	statusLabel := widget.NewLabel("Starting…")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	versionLabel := widget.NewLabel("")
	portLabel := widget.NewLabel("")
	devicesLabel := widget.NewLabel("Connected devices: —")
	deviceList := widget.NewLabel("")
	deviceList.Wrapping = fyne.TextWrapWord

	var (
		svcMu sync.Mutex
		svc   *bridge.Service
		stop  = make(chan struct{})
	)

	stopService := func() {
		svcMu.Lock()
		s := svc
		svc = nil
		svcMu.Unlock()
		if s != nil {
			_ = s.Stop()
		}
		select {
		case <-stop:
		default:
			close(stop)
		}
	}

	quitBtn := widget.NewButton("Quit", func() {
		w.Close()
	})
	quitBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("Trezor Bridge", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		versionLabel,
		portLabel,
		widget.NewSeparator(),
		devicesLabel,
		deviceList,
		widget.NewSeparator(),
		widget.NewLabel("Close this window to stop the bridge."),
		quitBtn,
	)
	w.SetContent(container.NewPadded(content))

	w.SetCloseIntercept(func() {
		stopService()
		w.Close()
	})

	go func() {
		const port = 21325
		opts := bridge.Options{
			Port:    port,
			WithUSB: true,
			Reset:   true,
			Version: bridge.Version,
			GitHash: bridge.ResolveGitHash(),
		}

		s, err := bridge.Start(opts)
		if err != nil && strings.Contains(err.Error(), "address already in use") {
			fyne.Do(func() {
				statusLabel.SetText("Taking over port…")
				versionLabel.SetText("Stopping the other Bridge so this app can start.")
			})
			if freeErr := bridge.FreePort(port); freeErr != nil {
				fyne.Do(func() {
					statusLabel.SetText("Failed to start")
					versionLabel.SetText(fmt.Sprintf("%v\n\nAlso could not free port: %v", err, freeErr))
					portLabel.SetText("")
					devicesLabel.SetText("")
				})
				return
			}
			s, err = bridge.Start(opts)
		}
		if err != nil {
			fyne.Do(func() {
				statusLabel.SetText("Failed to start")
				versionLabel.SetText(err.Error())
				portLabel.SetText("")
				devicesLabel.SetText("")
			})
			return
		}
		svcMu.Lock()
		svc = s
		svcMu.Unlock()

		fyne.Do(func() {
			statusLabel.SetText("Running")
			versionLabel.SetText(fmt.Sprintf("Version: %s (rev %s)", s.VersionString(), s.GitHash()))
			portLabel.SetText(fmt.Sprintf("Listening on http://127.0.0.1:%d", s.Port()))
		})

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		refreshDevices(s, devicesLabel, deviceList)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				refreshDevices(s, devicesLabel, deviceList)
			}
		}
	}()

	w.ShowAndRun()
	stopService()
}

func refreshDevices(s *bridge.Service, countLabel, listLabel *widget.Label) {
	entries, err := s.Enumerate()
	fyne.Do(func() {
		if err != nil {
			countLabel.SetText("Connected devices: error")
			listLabel.SetText(err.Error())
			return
		}
		countLabel.SetText(fmt.Sprintf("Connected devices: %d", len(entries)))
		if len(entries) == 0 {
			listLabel.SetText("No Trezor connected.")
			return
		}
		var b strings.Builder
		for i, e := range entries {
			if i > 0 {
				b.WriteString("\n")
			}
			session := "idle"
			if e.Session != nil {
				session = "session " + *e.Session
			}
			fmt.Fprintf(&b, "• %s — %s\n  %s", deviceTypeName(e.Type), session, e.Path)
		}
		listLabel.SetText(b.String())
	})
}

func deviceTypeName(t core.DeviceType) string {
	switch t {
	case core.TypeT1Hid:
		return "Trezor One (HID)"
	case core.TypeT1Webusb:
		return "Trezor One (WebUSB)"
	case core.TypeT1WebusbBoot:
		return "Trezor One (WebUSB, bootloader)"
	case core.TypeT2:
		return "Trezor Model T"
	case core.TypeT2Boot:
		return "Trezor Model T (bootloader)"
	case core.TypeEmulator:
		return "Trezor Emulator"
	default:
		return fmt.Sprintf("Device (type %d)", t)
	}
}

// bridgeTheme uses Trezor green accents on a light background.
type bridgeTheme struct{}

func (t *bridgeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x01, G: 0xB7, B: 0x57, A: 0xFF}
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xF7, G: 0xF7, B: 0xF7, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x01, G: 0xB7, B: 0x57, A: 0xFF}
	case theme.ColorNameForegroundOnPrimary:
		return color.White
	default:
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

func (t *bridgeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *bridgeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *bridgeTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
