// wifi is a daemon that tries joining a pre-configured WiFi network.
//
// Example:
//
//	Create a WiFi configuration file,
//	either via https://github.com/gokrazy/breakglass,
//	or by mounting the SD card on the host:
//	# echo '{"ssid": "I/O Tee"}' > /perm/wifi.json
//
//	Include the wifi package in your gokr-packer command:
//	% gokr-packer -update=yes \
//	  github.com/gokrazy/breakglass \
//	  github.com/gokrazy/wifi
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alf632/gokrazy-ha/wifi/iface"
	"github.com/gokrazy/gokrazy"
	"github.com/mdlayher/wifi"
	"golang.org/x/sys/unix"
)

type wifiConfig struct {
	SSID      string `json:"ssid"`
	PSK       string `json:"psk"`
	INTERFACE string `json:"interface"`
}

type wifiCtx struct {
	// config
	cl         *wifi.Client
	interfaces []*wifi.Interface
	cfg        *wifiConfig

	// state
	dhcpClientMu   sync.Mutex
	dhcpClient     *exec.Cmd
	connectedSince time.Duration
}

func (w *wifiCtx) control1() error {
Interface:
	for _, intf := range w.interfaces {
		log.Println("getting station info for", intf.Name)
		stationInfos, err := w.cl.StationInfo(intf)
		if err != nil && !errors.Is(err, os.ErrNotExist) /* not connected */ {
			log.Println("not connected")
			return err
		}
		for _, sta := range stationInfos {
			if bytes.Equal(sta.HardwareAddr, net.HardwareAddr{}) {
				log.Println("empty macaddress")
				continue
			}
			w.dhcpClientMu.Lock()
			log.Printf("connected to %v for %v, signal %v",
				sta.HardwareAddr,
				sta.Connected,
				sta.Signal)
			if sta.Connected < w.connectedSince {
				// reconnected. restart dhcp client
				if w.dhcpClient.Process != nil {
					w.dhcpClient.Process.Kill()
				}
				w.dhcpClient = nil
			}
			if w.dhcpClient != nil {
				w.dhcpClientMu.Unlock()
				continue Interface
			}
			w.dhcpClient = exec.Command("/gokrazy/dhcp", "-interface="+w.cfg.INTERFACE)
			w.dhcpClient.SysProcAttr = &syscall.SysProcAttr{
				// When the wifi process dies, make the kernel send a SIGTERM to
				// the dhcp process, too. The bake CI test runner uses
				// exec.CommandContext("wifi") which sends SIGKILL, so trying to
				// clean up the dhcp process from within wifi is fruitless.
				Pdeathsig: syscall.SIGTERM,
			}
			w.dhcpClient.Stdout = os.Stdout
			w.dhcpClient.Stderr = os.Stderr
			log.Printf("starting %v", w.dhcpClient.Args)
			w.dhcpClient.Start()
			go func() {
				w.dhcpClientMu.Lock()
				dhcpClient := w.dhcpClient
				w.dhcpClientMu.Unlock()
				if err := dhcpClient.Wait(); err != nil {
					log.Printf("dhcp process failed: %v", err)
				}
				w.dhcpClientMu.Lock()
				w.dhcpClient = nil
				w.dhcpClientMu.Unlock()
			}()
			w.dhcpClientMu.Unlock()
			continue Interface
		}

		// disconnected, ensure dhcp client is stopped:
		w.dhcpClientMu.Lock()
		if w.dhcpClient != nil && w.dhcpClient.Process != nil {
			w.dhcpClient.Process.Kill()
		}
		w.dhcpClient = nil
		w.dhcpClientMu.Unlock()

		log.Println("connecting...")
		// Interface is not associated with station, try connecting:
		if w.cfg.PSK != "" {
			if err := w.cl.ConnectWPAPSK(intf, w.cfg.SSID, w.cfg.PSK); err != nil {
				// -EALREADY means already connected, but misleadingly
				// stringifies to “operation already in progress”
				log.Printf("could not connect: %v", err)
			} else {
				log.Printf("connecting to SSID %q...", w.cfg.SSID)
			}
		} else {
			if err := w.cl.Connect(intf, w.cfg.SSID); err != nil {
				// -EALREADY means already connected, but misleadingly
				// stringifies to “operation already in progress”
				log.Printf("could not connect: %v", err)
			} else {
				log.Printf("connecting to SSID %q...", w.cfg.SSID)
			}
		}
	}
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
	return nil
}

func logic() error {
	var (
		disconnect = flag.Bool("disconnect",
			false,
			"instead of connecting to a WiFi network, disconnect the interface and exit")

		ssid = flag.String("ssid",
			"",
			"if non-empty, the ssid of the WiFi network to connect to. if empty, /perm/wifi.json or /etc/wifi.json will be used")

		psk = flag.String("psk",
			"",
			"if non-empty, the psk of the WiFi network to connect to. if empty, /perm/wifi.json or /etc/wifi.json will be used")
	)
	flag.Parse()
	var cfg wifiConfig
	if *ssid != "" || *disconnect {
		cfg.SSID = *ssid
		cfg.PSK = *psk
	} else {
		b, err := ioutil.ReadFile("/perm/wifi.json")
		if err != nil && os.IsNotExist(err) {
			b, err = ioutil.ReadFile("/etc/wifi.json")
		}
		if err != nil {
			if os.IsNotExist(err) {
				// No config file? Nothing to do!
				gokrazy.DontStartOnBoot()
			}
			return err
		}

		if err := json.Unmarshal(b, &cfg); err != nil {
			return err
		}
	}

	// modprobe the brcmfmac driver
	for _, mod := range []string{
		"kernel/drivers/net/wireless/broadcom/brcm80211/brcmutil/brcmutil.ko",
		"kernel/drivers/net/wireless/broadcom/brcm80211/brcmfmac/brcmfmac.ko",
		"kernel/drivers/net/wireless/broadcom/brcm80211/brcmfmac/wcc/brcmfmac-wcc.ko",
	} {
		if err := loadModule(mod); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	cl, err := wifi.New()
	if err != nil {
		return err
	}
	interfaces, err := cl.Interfaces()
	if err != nil {
		return err
	}
	if len(interfaces) == 0 {
		return fmt.Errorf("no interfaces found")
	}

	if *disconnect {
		for _, intf := range interfaces {
			if err := cl.Disconnect(intf); err != nil {
				return err
			}
		}
		return nil
	}

	w := &wifiCtx{
		cl:         cl,
		interfaces: interfaces,
		cfg:        &cfg,
	}

	cs, err := iface.NewConfigSocket(cfg.INTERFACE)
	if err != nil {
		return fmt.Errorf("config socket: %v", err)
	}
	defer cs.Close()

	b, err := ioutil.ReadFile("/sys/class/net/" + cfg.INTERFACE + "/address")
	if err != nil {
		return fmt.Errorf("reading /sys/class/net/"+cfg.INTERFACE+"/address: %v", err)
	}
	log.Printf("%s MAC address is %s", cfg.INTERFACE, strings.TrimSpace(string(b)))

	// Ensure the interface is up so that we can send DHCP packets.
	if err := cs.Up(); err != nil {
		log.Printf("setting link "+cfg.INTERFACE+" up: %v", err)
	}

	const controlLoopFrequency = 15 * time.Second
	for {
		if err := w.control1(); err != nil {
			log.Printf("control1: %v", err)
		}
		time.Sleep(controlLoopFrequency)
	}
	return nil
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
