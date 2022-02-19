package main

import (
	"flag"
	"fmt"
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
var clientPort = flag.String("cp", "8830", "client port")
var serverPort = flag.String("sp", "8829", "server port")

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
	m.Uncheck()
	doBtOpRepeat("disconnect", mac)
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

	if doBtOpRepeat("connect", mac) {
		m.Enable()
		m.Check()
		setBtAudio()
	} else {
		m.Enable()
		m.Uncheck()
	}
}

func doBtOpRepeat(arg ...string) bool {
	fmt.Println("~~ bt op:", arg)
	for i := 0; i < 10; i++ {
		_, err := doBtOp(arg...)
		if err == nil {
			return true
		}
		time.Sleep(800 * time.Millisecond)
	}
	return false
}

func doBtOp(arg ...string) (string, error) {
	cmd := exec.Command("bluetoothctl", arg...)
	stdout, err := cmd.Output()
	return string(stdout), err
}

func setBtAudio() {
	cmd := exec.Command("pactl", "list", "short", "sinks")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		sink := regexp.MustCompile(`bluez_sink\S+`).FindStringSubmatch(line)
		if len(sink) > 0 {
			fmt.Println("++ setting default audio to", sink)
			cmd := exec.Command("pactl", "set-default-sink", sink[0])
			_, err := cmd.Output()
			if err != nil {
				fmt.Println("fail set default bt audio", err)
			}
		}
	}
}

func startServer(serverPort string) {
	pc, err := net.ListenPacket("udp4", ":"+serverPort)
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

		fmt.Println("++ receiving query - wanted", req) //, string(buf[:n]))

		if requester == hostname {
			continue
		}

		localMtx.Lock()
		m, ok := localEndpoints[queriedMac]
		if ok && m.Checked() {
			disconnect(queriedMac, m) // someone wants to take over that device, we drop it
		}
		localMtx.Unlock()
	}
}

func pushNetwork(payload string) {
	for _, ip := range getBroadcasts() {
		fmt.Println("++ sending", payload)
		client, err := net.ListenPacket("udp4", ":"+*clientPort)
		if err != nil {
			fmt.Println("failed nw push", err)
			return
		}

		serverAddr, err := net.ResolveUDPAddr("udp4", ip+":"+*serverPort)
		if err != nil {
			fmt.Println("failed nw push", err)
			return
		}

		_, err = client.WriteTo([]byte(payload), serverAddr)
		if err != nil {
			fmt.Println("failed nw push", err)
			return
		}
	}
}

func startUI(uiReady chan bool) {
	// recheck sometimes

	onReady := func() {
		systray.SetIcon(Icon)
		systray.SetTitle("")
		systray.SetTooltip("")

		m := systray.AddMenuItem("quit", "quit")
		systray.AddSeparator()

		go func() {
			for {
				<-m.ClickedCh
				panic("quit") // classy
			}
		}()

		uiReady <- true
	}

	systray.Run(onReady, nil)
}

func scanPairedDevices() {
	fmt.Println("~~ scanning for avaiable devices")

	output, _ := doBtOp("devices")
	devices := strings.Split(output, "\n")

	localMtx.Lock()
	defer localMtx.Unlock()

	for _, device := range devices[:len(devices)-1] {
		infos := strings.SplitN(device, " ", 3)
		mac, name := infos[1], infos[2]

		output, _ := doBtOp("info", mac)
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
		panic("cant determine broadcast IP")
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
	flag.Parse()
	fmt.Printf("~~ bluebao starting\n\n")

	doBtOp("power", "on")

	uiReady := make(chan bool)
	go startUI(uiReady)
	<-uiReady
	go startServer(*serverPort)
	scanPairedDevices()

	select {}
}
