package status

import (
	"html/template"

	"github.com/trezor/trezord-go/api"
)

type statusTemplateDevice struct {
	Type    api.DeviceType
	Path    string
	Used    bool
	Session string
}

type statusTemplateData struct {
	Version     string
	Devices     []statusTemplateDevice
	DeviceCount int
	Log         string

	IsError   bool
	IsWindows bool
	Error     string

	CSRFField template.HTML
}

const templateString = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
  <title>Trezor Bridge status</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Helvetica Neue", Arial, sans-serif;
    }

    h1 {
      font-size: 36px;
    }

    p {
      color: #858585;
    }

    #container {
      width: 100%;
    }

    .error {
      border: 1px solid orangered;
      border-radius: 4px;
      min-width: 320px;
      max-width: 500px;
      min-height: 33px;
      margin: 20px auto;
      position: relative;
      color: darkred;
      padding-top: 13px;
    }

    .item {
      border: 1px solid lightgray;
      border-radius: 4px;
      min-width: 320px;
      max-width: 500px;
      min-height: 100px;
      margin: 20px auto;
      position: relative;
    }

    .item h3 {
      left: 20px;
      position: absolute;
    }

    .item p {
      top: 50px;
      left: -5px;
      position: relative;
      font-size: 11px;
    }

    .item .session {
      top: 20px;
      right: 20px;
      position: absolute;
    }

    .item-content {
      width: 100%;
    }

    .inner-container {
      max-width: 1024px;
      margin: 0 auto;
      text-align: center;
      border-radius: 4px;
    }

    .badge {
      display: inline-block;
      padding: 6px 10px 6px 10px;
      border: 1px solid #01B757;
      border-radius: 4px;
      color: #01B757;
    }

    .heading {
      margin-bottom: 40px;
    }

    .space-top {
      margin-top: 34px;
    }

    .btn-primary {
      display: inline-block;
      padding: 10px 40px 10px 40px;
      background-color: #01B757;
      color: white;
      border-radius: 4px;
    }

    .btn-primary:hover {
      background-color: #00A24C;
    }

    textarea{
      max-width: 700px;
    }

    #dlog {
      display: none;
    }

    /*fake link*/
    button {
      background: none !important;
      color: #069;
      border: none;
      padding: 0 !important;
      font: inherit;
      border-bottom: 1px solid #444;
      cursor: pointer;
    }
  </style>
</head>

<body>
  <div id="container">
    <div class="inner-container">
      <div class="heading">
        <h1>Trezor Bridge status</h1>
        <span class="badge">Version: {{.Version}}</span>
      </div>

      <p>Connected devices: {{.DeviceCount}}</p>

      {{if .IsError}}
        <div class="error">
          <b>Error:</b> {{.Error}}
        </div>
      {{end}}

      {{range .Devices}}
      <div class="item">
        <h3>{{if eq .Type 0}}
          Trezor One (HID)
        {{end}}

        {{if eq .Type 1}}
          Trezor One (WebUSB)
        {{end}}

        {{if eq .Type 2}}
          Trezor One (WebUSB, bootloader)
        {{end}}

        {{if eq .Type 3}}
          Trezor Model T
        {{end}}

        {{if eq .Type 4}}
          Trezor Model T (bootloader)
        {{end}}

        {{if eq .Type 5}}
          Trezor Emulator
        {{end}}

      </h3>
        <span class="session">
        {{if .Used}} Session: {{.Session}} {{end}} {{if not .Used}} Session: no session {{end}}
        </span>
        <p>Path: {{.Path}}</p>
       </div>
      {{end}}

       <div class="space-top">
       <p>Console Log
       </p>
       <textarea rows="25" cols="150" id="log">
{{.Log}}
       </textarea>
       <form>
         {{.CSRFField}}
         <a href="#" id="submitlog" onClick="doSubmit()">
           <div class="btn-primary">Download detailed log</div>
         </a>
         <div id="wait" class="badge" style="display: none">Please wait...</div>
         {{if .IsWindows}}
           <p style="margin-top: 6px">
              Detailed log might take a while to generate.
              <br>
              It might also reveal detailed information about your PC configuration.
           </p>
         {{end}}
       </form>
     </div>

      <div class="space-top">
        <p>You may need to reload the page after connecting / disconnecting device</p>
        <a href="#" onClick="location.href=location.href">
          <div class="btn-primary">Refresh page</div>
        </a>
      </div>
    </div>
  </div>
  <script>
  function doSubmit() {
    document.getElementById("submitlog").style.display = "none";
    document.getElementById("wait").style.display = "inline";

    // the time estimate is 100% fake
    // but we want to show user something so he feels
    // something is hapenning
    var run = true;
    var time = 90
    function runOne() {
        document.getElementById("wait").innerText = "Please wait " + time + " seconds";
        time -= 1
        if (time > 0 && run) {
            setTimeout(runOne, 1*1000)
        }
    }
    runOne()

    const formElement = document.getElementsByTagName("form")[0]
    const data = new URLSearchParams();
    for (const pair of new FormData(formElement)) {
      data.append(pair[0], pair[1]);
    }

    fetch("/status/log.gz", {
      method: 'post',
      body: data,
      credentials: 'same-origin',
    }).then(function(resp) {
      return resp.blob();
    }).then(function(blob) {
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");

      document.body.appendChild(a);
      a.style = "display: none";
      a.href = url;
      a.download = "log.gz";
      a.click();

      window.URL.revokeObjectURL(url);

      document.getElementById("submitlog").style.display = "inline";
      document.getElementById("wait").style.display = "none";
      run = false;
    });
  }
  </script>
</body>
</html>
`

var statusTemplate, _ = template.New("status").Parse(templateString)
