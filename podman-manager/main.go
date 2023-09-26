package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/gokrazy/gokrazy"
)

type PodmanInstance struct {
	name string
	image string
	tag string
	hostNetwork bool
	privileged bool
	volumes []string

	running bool
}

func newPodmanInstance(name, image, tag, volumesStr string, hostNetwork, privileged bool) &PodmanInstance {
	volumes := volumesStr.Split(",")
	return &PodmanInstance{name=name, image=image, tag=tag, hostNetwork=hostNetwork, privileged=privileged, volumes=volumes}
}

func (pi PodmanInstance) run() error {
	startArgs = []string{"run", "-td"}
	for volume := range pi.volumes {
		startArgs = append(startArgs, ["-v", volume])
	}
	if pi.hostNetwork {
		startArgs = append(startArgs, ["--network", "host"])
	}
	if pi.privileged {
		startArgs = append(startArgs, "--privileged")
	}
	startArgs = append(startArgs, ["--name", pi.name, pi.image+":"+pi.tag])
	podman(startArgs);
}

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

type PodmanManager struct {
	Instances map[string]PodmanInstance
}

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

func node-red() error {
	// Ensure we have an up-to-date clock, which in turn also means that
	// networking is up. This is relevant because podman takes what’s in
	// /etc/resolv.conf (nothing at boot) and holds on to it, meaning your
	// container will never have working networking if it starts too early.
	gokrazy.WaitForClock()

	if err := podman("build",
		"-t", "gokrazy-node-red:latest",
		"$GOPATH/pkg/mod/github.com/alf632/gokrazy-ha/node-red*/"
	); err != nil {
                return err
        }

	if err := mountVar(); err != nil {
		return err
	}

	if err := podman("kill", "node-red"); err != nil {
		log.Print(err)
	}

	if err := podman("rm", "node-red"); err != nil {
		log.Print(err)
	}


	if err := podman("run",
		"-td",
		"-v", "/perm/node-red:/config",
		"-v", "/etc/localtime:/etc/localtime:ro",
		"--network", "host",
		"--privileged",
		"--name", "node-red",
		"gokrazy-node-red:latest"); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := node-red(); err != nil {
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

