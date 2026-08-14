//go:build !darwin

package bridge

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// FreePort stops another process listening on the Bridge TCP port.
func FreePort(port int) error {
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
		return fmt.Errorf("port %d still in use by pid(s) %v (%s)", port, pids, runtime.GOOS)
	}
	return nil
}

func listenerPIDs(port int) ([]int, error) {
	out, err := exec.Command("lsof", "-t", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		// Windows may not have lsof; best-effort only
		return nil, nil
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
		if pid == os.Getpid() {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}
