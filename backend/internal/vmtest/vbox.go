//go:build vm

package vmtest

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

func vbox(args ...string) error {
	cmd := exec.Command("VBoxManage", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("VBoxManage %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func vmState(name string) (string, error) {
	out, err := exec.Command("VBoxManage", "showvminfo", name, "--machinereadable").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "VMState=") {
			return strings.Trim(strings.TrimPrefix(line, "VMState="), `"`), nil
		}
	}
	return "", fmt.Errorf("VMState not found for %q", name)
}

// PrepareVM restores snapshot and starts the VM headless.
func PrepareVM(cfg VMConfig) error {
	st, err := vmState(cfg.VMName)
	if err != nil {
		return err
	}
	switch st {
	case "running":
		_ = vbox("controlvm", cfg.VMName, "poweroff")
		time.Sleep(2 * time.Second)
	case "paused":
		_ = vbox("controlvm", cfg.VMName, "resume")
		_ = vbox("controlvm", cfg.VMName, "poweroff")
		time.Sleep(2 * time.Second)
	}
	if err := vbox("snapshot", cfg.VMName, "restore", cfg.VMSnapshot); err != nil {
		return err
	}
	return vbox("startvm", cfg.VMName, "--type", "headless")
}

// StopVM powers off the VM (best-effort).
func StopVM(name string) {
	st, err := vmState(name)
	if err != nil || st != "running" {
		return
	}
	_ = vbox("controlvm", name, "acpipowerbutton")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		st, _ = vmState(name)
		if st != "running" {
			return
		}
		time.Sleep(time.Second)
	}
	_ = vbox("controlvm", name, "poweroff")
}

// WaitRouter blocks until MikroTik api-ssl accepts TCP connections.
func WaitRouter(host string, port int, timeout time.Duration) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		last = err
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("router not reachable at %s: %v", addr, last)
}
