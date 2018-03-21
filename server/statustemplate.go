package server

import "html/template"

type statusTemplateDevType int

const (
	typeT1     statusTemplateDevType = 0
	typeT2     statusTemplateDevType = 1
	typeT2Boot statusTemplateDevType = 2
)

type statusTemplateDevice struct {
	Type    statusTemplateDevType
	Path    string
	Used    bool
	Session string
}

type statusTemplateData struct {
	Version     string
	Devices     []statusTemplateDevice
	DeviceCount int
	Log         string
}

const templateString = `
<html>
<head><title>TREZOR Bridge status</title></head>
<body>
<h3>TREZOR Bridge seems to be working</h3>
Version: {{.Version}}
<br><br>
Device count: {{.DeviceCount}}
<br><br>
Devices:
<br>
<br>
{{range .Devices}}
  
  <b>
  {{if eq .Type 0}}
    TREZOR One
  {{end}}

  {{if eq .Type 1}}
    TREZOR Model T
  {{end}}
  {{if eq .Type 2}}
    TREZOR Model T (bootloader)
  {{end}}
  </b>

  <br>
  {{.Path}}
  <br>

  {{if .Used}}
    Session: {{.Session}}
  {{end}}
  {{if not .Used}}
    Session: no session
  {{end}}
  <br><br>

{{end}}
<textarea rows="25" cols="80">
{{.Log}}
</textarea>
(You need to reload the page after connecting/disconnecting)
</body>
</html>
`

var statusTemplate, _ = template.New("status").Parse(templateString)
