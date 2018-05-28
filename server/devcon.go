// +build windows

package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/trezor/trezord-go/usb"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
)

func devconAllStatusInfo() (string, error) {
	_, err := os.Stat("devcon.exe")
	if os.IsNotExist(err) {
		return "devcon.exe does not exist\n", nil
	}
	if err != nil {
		return "", err
	}

	conn, disconn, err := devconAllUsbStrings()
	if err != nil {
		return "", err
	}

	res := "Status log\nConnected devices:\n"
	cm, err := devconMultipleStatuses(conn)
	if err != nil {
		return "", err
	}
	res += cm

	res += "\nDisonnected devices:\n"

	dm, err := devconMultipleStatuses(disconn)
	if err != nil {
		return "", err
	}
	res += dm

	res += "\n"
	return res, nil
}

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
	conn, disconn, err := devconTrezorUsbStrings(dlogger)
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

func devconAllUsbStrings() ([]string, []string, error) {
	all, err := devconUsbStringsEvery(true)

	if err != nil {
		return nil, nil, err
	}

	conn, err := devconUsbStringsEvery(false)
	if err != nil {
		return nil, nil, err
	}

	disconn := stringsDifference(all, conn)

	return conn, disconn, nil
}

func devconTrezorUsbStrings(dlogger *log.Logger) ([]string, []string, error) {
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
	disconn := stringsDifference(all, conn)

	return conn, disconn, nil
}

func stringsDifference(all, connected []string) []string {
	connMap := make(map[string]bool)
	for _, i := range connected {
		connMap[i] = true
	}

	disconnected := make([]string, 0, len(all)-len(connected))
	for _, i := range all {
		if !(connMap[i]) {
			disconnected = append(disconnected, i)
		}
	}
	return disconnected
}

func devconMultipleStatuses(ids []string) (string, error) {
	res := ""
	for _, i := range ids {
		s, err := devconStatus(i)
		if err != nil {
			return "", err
		}
		res += s + "\n\n"
	}
	return res, nil
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

func runDevcon(cmd, par string, dlogger *log.Logger, unicode bool) (string, error) {

	if dlogger != nil {
		dlogger.Println("devconInfo - runninng %s %s %s", "devcon.exe", cmd, par)
	}
	cmdInstance := exec.Command("devcon.exe", "-u", cmd, par) // nolint: gas
	if !unicode {
		cmdInstance = exec.Command("devcon.exe", cmd, par) // nolint: gas
	}
	cmdInstance.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmdInstance.Output()

	if err != nil {
		return "", err
	}

	contentStr := string(output)
	if unicode {
		contentStr = utf16BytesToString(output, binary.LittleEndian)
	}

	contentStr = strings.Replace(contentStr, "\r\n", "\n", -1)

	return contentStr, nil
}

func runMsinfo() (string, error) {
	windir := os.Getenv("windir") + "\\system32\\"

	tmpfile, err := ioutil.TempFile("", "trezorMsInfo")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	if err != nil {
		return "", err
	}

	cmdInstance := exec.Command(windir+"msinfo32.exe", "/report", tmpfile.Name())
	cmdInstance.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cc, err := cmdInstance.CombinedOutput()

	if err != nil {
		return "", errors.New(string(cc))
	}

	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	contentStr := utf16BytesToString(content, binary.LittleEndian)
	contentStr = strings.Replace(contentStr, "\r\n", "\n", -1)
	return contentStr, nil
}

func utf16BytesToString(b []byte, o binary.ByteOrder) string {
	utf := make([]uint16, (len(b)+(2-1))/2)
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = o.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))
}

func devconDriverFiles(id string, dlogger *log.Logger) (string, error) {
	dlogger.Println("devconInfo - finding driver files for %s", id)
	out, err := runDevcon("driverfiles", "@"+id, dlogger, false)
	if err != nil {
		return "", err
	}

	lines := strings.Split(out, "\n")
	lines = lines[0 : len(lines)-2]
	res := strings.Join(lines, "\n")
	return res, nil
}

func devconStatus(id string) (string, error) {
	out, err := runDevcon("status", "@"+id, nil, true)
	if err != nil {
		return "", err
	}

	lines := strings.Split(out, "\n")
	lines = lines[0 : len(lines)-2]
	res := id + "\n" + strings.Join(lines, "\n")
	return res, nil
}

func devconUsbStringsEvery(with_disconnected bool) ([]string, error) {
	return devconUsbStrings("*", with_disconnected, nil, true)
}

func devconUsbStrings(filter string, with_disconnected bool, dlogger *log.Logger, shit bool) ([]string, error) {
	command := "find"
	if with_disconnected {
		command = "findall"
	}
	out, err := runDevcon(command, filter, dlogger, false)

	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	if len(lines) == 2 {
		return nil, nil
	}

	lines = lines[0 : len(lines)-3]
	return lines, nil
}

func devconUsbStringsVid(vid int, with_disconnected bool, dlogger *log.Logger) ([]string, error) {
	v := fmt.Sprintf("*vid_%04x*", vid)
	return devconUsbStrings(v, with_disconnected, dlogger, false)
}
