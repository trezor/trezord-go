// +build windows

package server

import (
	"fmt"
	"github.com/trezor/trezord-go/usb"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func devconInfo(dlogger *log.Logger) (string, error) {
	dlogger.Println("devconInfo - finding devcon.exe")
	_, err := os.Stat("devcon.exe")
	if os.IsNotExist(err) {
		return "devcon.exe does not exist\n", nil
	}
	if err != nil {
		return "", err
	}

	dlogger.Println("devconInfo - usbStrings")
	conn, disconn, err := devconUsbStrings(dlogger)
	if err != nil {
		return "", err
	}

	res := "Driver log\nConnected devices:\n"
	dlogger.Println("devconInfo - finding driver files")
	cm, err := devconMultipleDriverFiles(conn, dlogger)
	if err != nil {
		return "", err
	}
	res += cm

	res += "\nDisonnected devices:\n"

	dm, err := devconMultipleDriverFiles(disconn, dlogger)
	if err != nil {
		return "", err
	}
	res += dm

	res += "\n"
	return res, nil
}

func devconUsbStrings(dlogger *log.Logger) ([]string, []string, error) {
	allT1, err := devconUsbStringsVid(usb.VendorT1, true, dlogger)
	if err != nil {
		return nil, nil, err
	}

	allT2, err := devconUsbStringsVid(usb.VendorT2, true, dlogger)
	if err != nil {
		return nil, nil, err
	}

	connT1, err := devconUsbStringsVid(usb.VendorT1, false, dlogger)
	if err != nil {
		return nil, nil, err
	}

	connT2, err := devconUsbStringsVid(usb.VendorT2, false, dlogger)
	if err != nil {
		return nil, nil, err
	}

	all := append(allT1, allT2...)
	conn := append(connT1, connT2...)

	connMap := make(map[string]bool)
	for _, i := range conn {
		connMap[i] = true
	}

	disconn := make([]string, 0, len(all)-len(conn))
	for _, i := range all {
		if !(connMap[i]) {
			disconn = append(disconn, i)
		}
	}
	return conn, disconn, nil
}

func devconMultipleDriverFiles(ids []string, dlogger *log.Logger) (string, error) {
	res := ""
	for _, i := range ids {
		driverFiles, err := devconDriverFiles(i, dlogger)
		if err != nil {
			return "", err
		}
		res += driverFiles + "\n\n"
	}
	return res, nil
}

func runDevcon(cmd, par string, dlogger *log.Logger) (string, error) {
	dlogger.Println("devconInfo - runninng %s %s %s", "devcon.exe", cmd, par)
	cmdInstance := exec.Command("devcon.exe", cmd, par) // nolint: gas
	cmdInstance.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmdInstance.Output()

	if err != nil {
		return "", err
	}
	return string(output), nil
}

func devconDriverFiles(id string, dlogger *log.Logger) (string, error) {
	dlogger.Println("devconInfo - finding driver files for %s", id)
	out, err := runDevcon("driverfiles", "@"+id, dlogger)
	if err != nil {
		return "", err
	}

	lines := strings.Split(out, "\r\n")
	lines = lines[0 : len(lines)-2]
	res := strings.Join(lines, "\n")
	return res, nil
}

func devconUsbStringsVid(vid int, all bool, dlogger *log.Logger) ([]string, error) {
	command := "find"
	if all {
		command = "findall"
	}
	v := fmt.Sprintf("*vid_%04x*", vid)
	out, err := runDevcon(command, v, dlogger)

	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\r\n")
	if len(lines) == 2 {
		return nil, nil
	}

	lines = lines[0 : len(lines)-3]
	return lines, nil
}
