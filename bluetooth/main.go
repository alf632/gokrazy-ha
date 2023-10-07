package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bitfield/script"

	"github.com/gokrazy/gokrazy"
	"golang.org/x/sys/unix"
)

func podman(args ...string) error {
	podman := exec.Command("/usr/local/bin/podman", args...)
	podman.Env = expandPath(os.Environ())
	podman.Env = append(podman.Env, "TMPDIR=/tmp")
	podman.Stdin = os.Stdin
	podman.Stdout = os.Stdout
	podman.Stderr = os.Stderr
	if err := podman.Run(); err != nil {
		return fmt.Errorf("%v: %v", podman.Args, err)
	}
	return nil
}

func checkImage(image string) bool {
	lines, err := script.Exec("/usr/local/bin/podman images").Match(image).CountLines()
	if err != nil {
		log.Print(err)
		return false
	}
	return lines > 0
}

func bluetooth() error {
	// Ensure we have an up-to-date clock, which in turn also means that
	// networking is up. This is relevant because podman takes what’s in
	// /etc/resolv.conf (nothing at boot) and holds on to it, meaning your
	// container will never have working networking if it starts too early.
	gokrazy.WaitForClock()

	if !checkImage("gokrazy-bluetooth") {

		if err := podman("build",
			"-t", "gokrazy-bluetooth:latest",
			"--no-cache",
			"https://raw.githubusercontent.com/alf632/gokrazy-ha/main/bluetooth/Dockerfile",
		); err != nil {
			return err
		}

	}

	if err := initBluetooth(); err != nil {
		return err
	}

	if err := mountVar(); err != nil {
		return err
	}

	if err := podman("kill", "bluetooth"); err != nil {
		log.Print(err)
	}

	if err := podman("rm", "bluetooth"); err != nil {
		log.Print(err)
	}

	if err := podman("run",
		"-td",
		"-v", "/etc/localtime:/etc/localtime:ro",
		"--network", "host",
		"--privileged",
		"--name", "bluetooth",
		"gokrazy-bluetooth:latest"); err != nil {
		return err
	}

	// gokrazy should not supervise this process even when manually started.
	os.Exit(125)

	return nil
}

func main() {
	if err := bluetooth(); err != nil {
		log.Fatal(err)
	}
}

// mountVar bind-mounts /perm/container-storage to /var if needed.
// This could be handled by an fstab(5) feature in gokrazy in the future.
func mountVar() error {
	b, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		log.Printf("Cannot Check mountpoint!")
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		mountpoint := parts[4]
		log.Printf("Found mountpoint %q", parts[4])
		if mountpoint == "/var" {
			log.Printf("/var file system already mounted, nothing to do")
			return nil
		}
	}

	if err := syscall.Mount("/perm/container-storage", "/var", "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("mounting /perm/container-storage to /var: %v", err)
	}

	return nil
}

// expandPath returns env, but with PATH= modified or added
// such that both /user and /usr/local/bin are included, which podman needs.
func expandPath(env []string) []string {
	extra := "/user:/usr/local/bin"
	found := false
	for idx, val := range env {
		parts := strings.Split(val, "=")
		if len(parts) < 2 {
			continue // malformed entry
		}
		key := parts[0]
		if key != "PATH" {
			continue
		}
		val := strings.Join(parts[1:], "=")
		env[idx] = fmt.Sprintf("%s=%s:%s", key, extra, val)
		found = true
	}
	if !found {
		const busyboxDefaultPATH = "/usr/local/sbin:/sbin:/usr/sbin:/usr/local/bin:/bin:/usr/bin"
		env = append(env, fmt.Sprintf("PATH=%s:%s", extra, busyboxDefaultPATH))
	}
	return env
}

func initBluetooth() error {
	// modprobe the hci_uart driver for Raspberry Pi (3B+, others)
	for _, mod := range []string{
		"kernel/crypto/ecc.ko",
		"kernel/crypto/ecdh_generic.ko",
		"kernel/net/bluetooth/bluetooth.ko",
		"kernel/drivers/bluetooth/btbcm.ko",
		"kernel/drivers/bluetooth/hci_uart.ko",
	} {
		if err := loadModule(mod); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	dev := "hci0"
	target, err := checkBluetoothInterface(dev)
	if err != nil {
		log.Printf("Bluetooth interface %v not found.", target)
	} else {
		fmt.Printf("Bluetooth device %v: %v\n", dev, target)
	}

	return nil
}

func checkBluetoothInterface(device string) (string, error) {
	target, err := os.Readlink("/sys/class/bluetooth/hci0")
	if err != nil {
		return "", fmt.Errorf("Bluetooth interface %v not found", device)
	}
	return target, nil
}

func loadModule(mod string) error {
	f, err := os.Open(filepath.Join("/lib/modules", release, mod))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := unix.FinitModule(int(f.Fd()), "", 0); err != nil {
		if err != unix.EEXIST &&
			err != unix.EBUSY &&
			err != unix.ENODEV &&
			err != unix.ENOENT {
			return fmt.Errorf("FinitModule(%v): %v", mod, err)
		}
	}
	modname := strings.TrimSuffix(filepath.Base(mod), ".ko")
	log.Printf("modprobe %v", modname)
	return nil
}

var release = func() string {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		fmt.Fprintf(os.Stderr, "minitrd: %v\n", err)
		os.Exit(1)
	}
	return string(uts.Release[:bytes.IndexByte(uts.Release[:], 0)])
}()
