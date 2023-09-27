package podmanManager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/bitfield/script"
	"github.com/gokrazy/gokrazy"
)

type PodmanInstance struct {
	name        string
	image       string
	tag         string
	hostNetwork bool
	privileged  bool
	volumes     []string

	buildContext string
}

func newPodmanInstance(name, image, tag, volumesStr string, hostNetwork, privileged bool) *PodmanInstance {
	volumes := strings.Split(volumesStr, ",")
	return &PodmanInstance{name: name, image: image, tag: tag, hostNetwork: hostNetwork, privileged: privileged, volumes: volumes}
}

func (pi PodmanInstance) checkImageExists() bool {
	lines, err := script.Exec("/usr/local/bin/podman images").Match(pi.image).CountLines()
	if err != nil {
		log.Print(err)
		return false
	}
	return lines > 0
}

func (pi PodmanInstance) build() error {
	if err := podman("build",
		"-t", pi.image+":"+pi.tag,
		pi.buildContext); err != nil {
		return err
	}
	return nil
}

func (pi PodmanInstance) run() error {
	gokrazy.WaitForClock()

	if !pi.checkImageExists() {
		pi.build()
	}

	if err := mountVar(); err != nil {
		return err
	}

	if err := podman("kill", pi.name); err != nil {
		log.Print(err)
	}

	if err := podman("rm", pi.name); err != nil {
		log.Print(err)
	}

	startArgs := []string{"run", "-td"}
	for _, volume := range pi.volumes {
		startArgs = append(startArgs, "-v", volume)
	}
	if pi.hostNetwork {
		startArgs = append(startArgs, "--network", "host")
	}
	if pi.privileged {
		startArgs = append(startArgs, "--privileged")
	}
	startArgs = append(startArgs, "--name", pi.name, pi.image+":"+pi.tag)

	if err := podman(startArgs...); err != nil {
		return err
	}

	if err := podman("logs", "-f", pi.name); err != nil {
		return err
	}
	return nil
}

// podman wraps the podman binary and redirects STDIO
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
