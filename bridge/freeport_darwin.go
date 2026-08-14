//go:build darwin

package bridge

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const macosLaunchAgentPlist = "/Library/LaunchAgents/com.bitcointrezor.trezorBridge.trezord.plist"

// FreePort stops another Bridge holding the TCP port so this app can bind it.
// On macOS this unloads the official Trezor Bridge LaunchAgent (KeepAlive),
// then terminates any remaining listener on the port.
func FreePort(port int) error {
	uid := os.Getuid()
	// Prefer bootout (modern launchctl); fall back to unload.
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d", uid), macosLaunchAgentPlist).Run()
	_ = exec.Command("launchctl", "unload", macosLaunchAgentPlist).Run()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		pids, err := listenerPIDs(port)
		if err != nil {
			return err
		}
		if len(pids) == 0 {
			return nil
		}
		for _, pid := range pids {
			_ = exec.Command("kill", strconv.Itoa(pid)).Run()
		}
		time.Sleep(200 * time.Millisecond)
	}

	pids, err := listenerPIDs(port)
	if err != nil {
		return err
	}
	if len(pids) > 0 {
		return fmt.Errorf("port %d still in use by pid(s) %v after stopping system Bridge", port, pids)
	}
	return nil
}

func listenerPIDs(port int) ([]int, error) {
	out, err := exec.Command("lsof", "-t", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		// lsof exits 1 when nothing matches
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		// Never kill ourselves
		if pid == os.Getpid() {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}
