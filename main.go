package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"encoding/json"
)

type endpoint struct {
	Onit string            `json:"onit"`
	Menu *systray.MenuItem `json:"-"`
}

// find endpoint per mac addess
var localEndpoints map[string]endpoint
var localStateMtx sync.Mutex
var hostname string

var client net.PacketConn
var clientAddr *net.UDPAddr

func addUIEntry(name string, mac string) *systray.MenuItem {
	m := systray.AddMenuItemCheckbox(name, name, false)

	go func() {
		for {
			<-m.ClickedCh
			localStateMtx.Lock()
			l := localEndpoints[mac]

			if m.Checked() {
				disconnect(mac, &l)
			} else {
				connect(mac, &l)
			}

			localEndpoints[mac] = l
			localStateMtx.Unlock()
			send()
		}
	}()

	return m
}

func merge(buf []byte) {
	remoteEndpoints := make(map[string]endpoint)
	errMarshall := json.Unmarshal(buf, &remoteEndpoints)
	if errMarshall != nil {
		fmt.Println("~~ failed at parsing remote state")
		return
	}

	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	for mac, r := range remoteEndpoints {
		l, ok := localEndpoints[mac]

		if !ok {
			continue
		}

		localConn := l.Onit == hostname
		shouldBeConned := r.Onit == hostname

		// remote instruction received
		if localConn && !shouldBeConned {
			disconnect(mac, &l)
		} else if !localConn && shouldBeConned {
			connect(mac, &l)
		}

		l.Onit = r.Onit
		localEndpoints[mac] = l
	}
}

func disconnect(mac string, t *endpoint) {
	t.Menu.Uncheck()

	if t.Onit == hostname {
		t.Onit = ""
	}

	doBtOp(0, "disconnect", mac)
}

func connect(mac string, t *endpoint) {
	t.Menu.Check()
	t.Onit = hostname

	// only 1 audio device allowed at the same time
	for n, l := range localEndpoints {
		if l.Onit == hostname {
			disconnect(mac, &l)
			localEndpoints[n] = l
		}
	}

	time.Sleep(800 * time.Millisecond)
	doBtOp(0, "connect", mac)
}

func doBtOp(cnt int, arg ...string) string {
	cmd := exec.Command("bluetoothctl", arg...)
	stdout, err := cmd.Output()
	fmt.Println("err", err, "stdout", string(stdout))
	if err != nil && cnt < 10 {
		time.Sleep(800 * time.Millisecond)
		return doBtOp(cnt+1, arg...)
	}
	return string(stdout)
}

func send() {
	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	jsonValue, err := json.Marshal(localEndpoints)
	if err != nil {
		fmt.Println("error marshal local")
		return
	}

	if string(jsonValue) == "{}" {
		return
	}

	fmt.Println("++ sending state")
	_, err = client.WriteTo(jsonValue, clientAddr)
	if err != nil {
		fmt.Println("err writing state", err)
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

		fmt.Println("++ receiving remote state") //, string(buf[:n]))
		merge(buf[:n])
	}
}

func startClient(multicastAddr string, clientPort string, serverPort string) {
	var err error
	client, err = net.ListenPacket("udp4", ":"+clientPort)
	if err != nil {
		panic(err)
	}

	clientAddr, err = net.ResolveUDPAddr("udp4", multicastAddr+":"+serverPort)
	if err != nil {
		panic(err)
	}
}

func startUI() {
	onReady := func() {
		systray.SetIcon(Icon)
		systray.SetTitle("")
		systray.SetTooltip("")

		systray.AddSeparator()
		m := systray.AddMenuItem("quit", "quit")
		go func() {
			for {
				<-m.ClickedCh
				panic("quit") // classy
			}
		}()
	}

	systray.Run(onReady, nil)
}

func scanPairedDevices() {
	fmt.Println("~~ scanning for avaiable devices")

	output := doBtOp(0, "devices")
	devices := strings.Split(output, "\n")

	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	for _, device := range devices[:len(devices)-1] {
		infos := strings.SplitN(device, " ", 3)
		mac, name := infos[1], infos[2]

		output := doBtOp(0, "info", mac)
		if strings.Contains(output, "Audio") {
			var ui = addUIEntry(name, mac)
			localEndpoints[mac] = endpoint{"", ui}
		}
	}
}

func main() {
	var multicastAddr = flag.String("h", "192.168.0.255", "multicast address")
	var clientPort = flag.String("cp", "8830", "client port")
	var serverPort = flag.String("sp", "8829", "server port")

	flag.Parse()

	doBtOp(0, "power", "on")

	hostname, _ = os.Hostname()
	localEndpoints = make(map[string]endpoint)

	go startUI()
	time.Sleep(100 * time.Millisecond)

	startClient(*multicastAddr, *clientPort, *serverPort)
	go startServer(*serverPort)
	fmt.Printf("~~ bluebao starting, name %s\n\n", hostname)
	scanPairedDevices()

	select {}
}
