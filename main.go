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
var localEndpoints map[string]*endpoint
var localStateMtx sync.Mutex
var hostname string

var client net.PacketConn
var clientAddr *net.UDPAddr

func addUIEntry(name string, mac string) *endpoint {
	m := systray.AddMenuItemCheckbox(name, name, false)
	entry := endpoint{"", m}

	go func() {
		for {
			<-m.ClickedCh
			localStateMtx.Lock()

			if m.Checked() {
				disconnect(true, mac, &entry)
			} else {
				connect(true, mac, &entry)
			}

			localStateMtx.Unlock()
		}
	}()

	return &entry
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
			disconnect(false, mac, l)
		} else if !localConn && shouldBeConned {
			connect(false, mac, l)
		}
	}
}

func disconnect(emit_state bool, mac string, t *endpoint) {
	t.Menu.Uncheck()
	t.Onit = ""
	if emit_state {
		send()
	}
	doBtOpRepeat("disconnect", mac)
}

func connect(emit_state bool, mac string, t *endpoint) {
	t.Menu.Check()

	// only 1 audio device allowed at the same time, disconnect others
	for otherMac, e := range localEndpoints {
		if otherMac != mac && e.Onit == hostname {
			disconnect(false, otherMac, e)
		}
	}

	t.Onit = hostname
	if emit_state {
		send()
	}
	doBtOpRepeat("connect", mac)
}

func doBtOpRepeat(arg ...string) (string, error) {
	var err error = nil
	var ret string
	for i := 0; i < 10; i++ {
		ret, err = doBtOp(arg...)
		if err == nil {
			break
		}
		time.Sleep(800 * time.Millisecond)
	}
	return ret, err
}

func doBtOp(arg ...string) (string, error) {
	cmd := exec.Command("bluetoothctl", arg...)
	stdout, err := cmd.Output()
	// fmt.Println("err", err, "stdout", string(stdout))
	return string(stdout), err
}

func send() {
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

	output, _ := doBtOp("devices")
	devices := strings.Split(output, "\n")

	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	for _, device := range devices[:len(devices)-1] {
		infos := strings.SplitN(device, " ", 3)
		mac, name := infos[1], infos[2]

		output, _ := doBtOp("info", mac)
		if strings.Contains(output, "Audio") {
			localEndpoints[mac] = addUIEntry(name, mac)
		}
	}
}

func main() {
	var multicastAddr = flag.String("h", "192.168.0.255", "multicast address")
	var clientPort = flag.String("cp", "8830", "client port")
	var serverPort = flag.String("sp", "8829", "server port")

	flag.Parse()
	fmt.Printf("~~ bluebao starting, name %s\n\n", hostname)

	doBtOp("power", "on")

	hostname, _ = os.Hostname()
	localEndpoints = make(map[string]*endpoint)

	go startUI()
	time.Sleep(100 * time.Millisecond)

	startClient(*multicastAddr, *clientPort, *serverPort)
	go startServer(*serverPort)

	scanPairedDevices()

	select {}
}
