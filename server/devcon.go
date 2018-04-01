// +build windows

package server

import (
	"fmt"
	"github.com/trezor/trezord-go/usb"
	"os"
	"os/exec"
	"strings"
)

func devconInfo() (string, error) {
	_, err := os.Stat("devcon.exe")
	if os.IsNotExist(err) {
		return "devcon.exe does not exist\n", nil
	}
	if err != nil {
		return "", err
	}

	conn, disconn, err := devconUsbStrings()
	if err != nil {
		return "", err
	}

	res := "Driver log\nConnected devices:\n"
	cm, err := devconMultipleDriverFiles(conn)
	if err != nil {
		return "", err
	}
	res += cm

	res += "\nDisonnected devices:\n"

	dm, err := devconMultipleDriverFiles(disconn)
	if err != nil {
		return "", err
	}
	res += dm

	res += "\n"
	return res, nil
}

func devconUsbStrings() ([]string, []string, error) {
	allT1, err := devconUsbStringsVid(usb.VendorT1, true)
	if err != nil {
		return nil, nil, err
	}

	allT2, err := devconUsbStringsVid(usb.VendorT2, true)
	if err != nil {
		return nil, nil, err
	}

	connT1, err := devconUsbStringsVid(usb.VendorT1, false)
	if err != nil {
		return nil, nil, err
	}

	connT2, err := devconUsbStringsVid(usb.VendorT2, false)
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

func devconMultipleDriverFiles(ids []string) (string, error) {
	res := ""
	for _, i := range ids {
		driverFiles, err := devconDriverFiles(i)
		if err != nil {
			return "", err
		}
		res += driverFiles + "\n\n"
	}
	return res, nil
}

func devconDriverFiles(id string) (string, error) {
	c := exec.Command("devcon.exe", "driverfiles", "@"+id) // nolint: gas
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\r\n")
	lines = lines[0 : len(lines)-2]
	res := strings.Join(lines, "\n")
	return res, nil
}

func devconUsbStringsVid(vid int, all bool) ([]string, error) {
	command := "find"
	vidHex := fmt.Sprintf("%04x", vid)
	if all {
		command = "findall"
	}
	c := exec.Command("devcon.exe", command, "*vid_"+vidHex+"*") // nolint: gas
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\r\n")
	if len(lines) == 2 {
		return nil, nil
	}

	lines = lines[0 : len(lines)-3]
	return lines, nil
}
