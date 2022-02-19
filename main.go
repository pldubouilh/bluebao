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
	Mac  string            `json:"mac"`
	Onit string            `json:"onit"`
	Menu *systray.MenuItem `json:"-"`
}

var localEndpoints map[string]endpoint
var localStateMtx sync.Mutex
var localName string

var client net.PacketConn
var clientAddr *net.UDPAddr

func addUIEntry(name string) *systray.MenuItem {
	m := systray.AddMenuItemCheckbox(name, name, false)

	go func() {
		for {
			<-m.ClickedCh
			localStateMtx.Lock()
			l := localEndpoints[name]

			if m.Checked() {
				disconnect(&l)
			} else {
				connect(&l)
			}

			localEndpoints[name] = l
			localStateMtx.Unlock()
			send()
		}
	}()

	return m
}

func mergeBytes(buf []byte) {
	remoteEndpoints := make(map[string]endpoint)
	errMarshall := json.Unmarshal(buf, &remoteEndpoints)
	if errMarshall != nil {
		fmt.Println("~~ failed at parsing remote state")
		return
	}

	merge(remoteEndpoints)
}

func merge(remoteEndpoints map[string]endpoint) {
	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	for name, r := range remoteEndpoints {
		l, ok := localEndpoints[name]
		l.Mac = r.Mac

		localConn := l.Onit == localName
		shouldBeConned := r.Onit == localName

		if !ok {
			fmt.Println("~~ new sink", name)
			l.Menu = addUIEntry(name)
		}

		if localConn && !shouldBeConned {
			disconnect(&l)
		} else if !localConn && shouldBeConned {
			connect(&l)
		}

		l.Onit = r.Onit
		localEndpoints[name] = l
	}
}

func disconnect(t *endpoint) {
	t.Menu.Uncheck()

	if t.Onit == localName {
		t.Onit = ""
	}

	doBtOp(0, "disconnect", t.Mac)
}

func connect(t *endpoint) {
	t.Menu.Check()
	t.Onit = localName

	for n, l := range localEndpoints {
		if l.Onit == localName {
			disconnect(&l)
			localEndpoints[n] = l
		}
	}

	time.Sleep(800 * time.Millisecond)
	doBtOp(0, "connect", t.Mac)
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
		mergeBytes(buf[:n])
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
		systray.SetTitle("bluebao")
		systray.SetTooltip("bluebao")

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
	pairedDevices := make(map[string]endpoint)

	for _, device := range devices[:len(devices)-1] {
		infos := strings.SplitN(device, " ", 3)
		mac, name := infos[1], infos[2]
		pairedDevices[name] = endpoint{mac, "", nil}
	}

	merge(pairedDevices)
}

func main() {
	var multicastAddr = flag.String("h", "192.168.0.255", "multicast address")
	var clientPort = flag.String("cp", "8830", "client port")
	var serverPort = flag.String("sp", "8829", "server port")

	flag.Parse()

	localName, _ = os.Hostname()
	localEndpoints = make(map[string]endpoint)

	go startUI()
	time.Sleep(100 * time.Millisecond)
	startClient(*multicastAddr, *clientPort, *serverPort)
	go startServer(*serverPort)

	fmt.Printf("~~ bluebao starting, name %s\n\n", localName)
	scanPairedDevices()

	select {}
}
