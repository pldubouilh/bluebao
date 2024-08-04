package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

// mac address // menu
var localEndpoints = make(map[string]*systray.MenuItem)
var localMtx sync.Mutex

var hostname, _ = os.Hostname()
var serverPort = flag.String("sp", "8829", "server port")
var enableNetwork = flag.Bool("d", false, "disable network feature")

func addUIEntry(name string, mac string) *systray.MenuItem {
	m := systray.AddMenuItemCheckbox(name, name, false)

	go func() {
		for {
			<-m.ClickedCh
			localMtx.Lock()

			if m.Checked() {
				disconnect(mac, m)
			} else {
				connect(mac, m)
			}

			localMtx.Unlock()
		}
	}()

	return m
}

func disconnect(mac string, m *systray.MenuItem) {
	go setDefaultAudio("Headphones")
	if btOptOutOk("disconnect", mac) {
		m.Uncheck()
	}
}

func connect(mac string, m *systray.MenuItem) {
	m.Disable()
	pushNetwork(hostname + "," + mac)

	// only 1 audio device allowed at the same time, disconnect others
	for macaddr, menu := range localEndpoints {
		if menu.Checked() {
			disconnect(macaddr, menu)
		}
	}

	// allow for network propagation
	time.Sleep(200 * time.Millisecond)

	if btOptOutOk("connect", mac) {
		m.Check()
		go setDefaultAudio("bluez")
	}

	m.Enable()
}

func btOptOut(arg ...string) (string, error) {
	cmd := exec.Command("bluetoothctl", arg...)
	stdout, err := cmd.Output()
	fmt.Println("> bluetoothctl", arg)
	fmt.Println("<", string(stdout), err)
	return string(stdout), err
}

func btOptOutOk(arg ...string) bool {
	_, err := btOptOut(arg...)
	return err == nil
}

func find(input string, entryType string) *string {
	type output struct {
		Name string `json:"name"`
		// other fields are ignored
	}

	for i := 0; i < 20; i++ {
		c := exec.Command("pactl", "-f", "json", "list", "short", entryType)
		stdout, _ := c.Output()
		var out []output
		if err := json.Unmarshal(stdout, &out); err != nil {
			log.Printf("Error parsing JSON: %v\n", err)
			continue
		}

		for _, s := range out {
			if strings.Contains(s.Name, input) {
				return &s.Name
			}
		}

		time.Sleep(200 * time.Millisecond)
		continue // retry
	}

	log.Printf("Failed to find %s\n", input)
	return nil
}

func setDefaultAudio(input string) {
	fmt.Println("trying to set default audio to", input)
	sink := find(input, "sinks")
	if sink == nil {
		return
	}

	fmt.Println("~~ setting default audio to", sink)
	cmd := exec.Command("pactl", "set-default-sink", *sink)
	_, err := cmd.Output()
	if err != nil {
		fmt.Println("failed to set default bt audio", err)
	} else {
		fmt.Println("~~ default audio set to", sink)
	}
}

func setProfile(profile string) {
	card := find("bluez", "cards")
	if card == nil {
		return
	}
	cmd := exec.Command("pactl", "set-card-profile", *card, profile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Println("failed to set audio profile", err)
	}
}

func startServer() {
	if !*enableNetwork {
		return
	}

	pc, err := net.ListenPacket("udp4", ":"+*serverPort)
	if err != nil {
		panic(err)
	}
	defer pc.Close()

	for {
		buf := make([]byte, 9000)
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			fmt.Println("err reading server", err)
		}

		req := strings.SplitN(string(buf[:n]), ",", 2)
		requester, queriedMac := req[0], req[1]

		if requester == hostname {
			continue
		}

		// fmt.Println("~~ receiving query", req) //, string(buf[:n]))

		localMtx.Lock()
		m, ok := localEndpoints[queriedMac]
		if ok && m.Checked() {
			disconnect(queriedMac, m) // someone wants to take over that device, we drop it
		}
		localMtx.Unlock()
	}
}

func pushNetwork(payload string) {
	if !*enableNetwork {
		return
	}

	for _, ip := range getBroadcasts() {
		fmt.Println("~~ broadcasting", payload)
		conn, err := net.Dial("udp4", ip+":"+*serverPort)
		if err != nil {
			fmt.Println("failed nw push", err)
		}

		conn.Write([]byte(payload))
		conn.Close()
	}
}

func startUI(uiReady chan bool) {
	onReady := func() {
		systray.SetIcon(Icon)
		systray.SetTitle("")
		systray.SetTooltip("")

		menuQuit := systray.AddMenuItem("Quit", "Quit")
		audioProfile := systray.AddMenuItem("Audio profile", "Audio profile")
		menuHq := audioProfile.AddSubMenuItem("High Quality", "High Quality")
		menuHeadset := audioProfile.AddSubMenuItem("Headset + Microphone", "Headset + Microphone")

		systray.AddSeparator()

		go func() {
			for {
				<-menuQuit.ClickedCh
				systray.Quit()
			}
		}()

		go func() {
			for {
				<-menuHq.ClickedCh
				setProfile("a2dp-sink")
			}
		}()
		go func() {
			for {
				<-menuHeadset.ClickedCh
				setProfile("headset-head-unit")
			}
		}()

		uiReady <- true
	}

	systray.Run(onReady, nil)
}

func scanPairedDevices() {
	fmt.Println("~~ scanning for avaiable devices")

	output, _ := btOptOut("devices")
	devices := strings.Split(output, "\n")

	localMtx.Lock()
	defer localMtx.Unlock()

	for _, device := range devices[:len(devices)-1] {
		infos := strings.SplitN(device, " ", 3)
		mac, name := infos[1], infos[2]

		output, _ := btOptOut("info", mac)
		connected := strings.Contains(output, "Connected: yes")
		if strings.Contains(output, "Audio") {
			localEndpoints[mac] = addUIEntry(name, mac)
			if connected {
				localEndpoints[mac].Check()
			}
		}
	}
}

func getBroadcasts() []string {
	cmd := exec.Command("ip", "addr", "show")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println("cant determine broadcast IPs")
	}

	ips := make([]string, 0)
	for _, line := range strings.Split(string(stdout), "\n") {
		if strings.Contains(line, "brd") && strings.Contains(line, "inet") {
			re := regexp.MustCompile(`brd\s+([0-9\.]+)`)
			ip := re.FindStringSubmatch(line)[1]
			ips = append(ips, ip)
		}
	}

	return ips
}

func main() {
	flag.Usage = func() {
		fmt.Println("🥟 bluebao\nA simple bluetooth audio devices manager to easily manage multiple devices.")
		fmt.Println()
		flag.PrintDefaults()
	}

	flag.Parse()
	fmt.Println("~~ bluebao starting")
	btOptOut("power", "on")

	uiReady := make(chan bool)
	go startUI(uiReady)
	<-uiReady
	go startServer()
	scanPairedDevices()

	select {}
}
