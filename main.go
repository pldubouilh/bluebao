package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"encoding/json"
)

type endpoint struct {
	Macs      []string          `json:"macs"`
	Excluding string            `json:"excluding"`
	Onit      string            `json:"onit"`
	Menu      *systray.MenuItem `json:"-"`
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

func merge(buf []byte) {

	remoteEndpoints := make(map[string]endpoint)
	errMarshall := json.Unmarshal(buf, &remoteEndpoints)
	if errMarshall != nil {
		fmt.Println("~~ failed at parsing remote state")
		return
	}

	localStateMtx.Lock()
	defer localStateMtx.Unlock()

	for name, r := range remoteEndpoints {
		l, ok := localEndpoints[name]
		l.Macs = r.Macs
		l.Excluding = r.Excluding

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

	doBtOps("disconnect", t.Macs)
}

func connect(t *endpoint) {
	t.Menu.Check()
	t.Onit = localName

	for n, l := range localEndpoints {
		if l.Onit == localName && l.Excluding == t.Excluding {
			disconnect(&l)
			localEndpoints[n] = l
		}
	}

	time.Sleep(800 * time.Millisecond)
	doBtOps("connect", t.Macs)
}

func doBtOps(op string, macs []string) {
	for _, mac := range macs {
		doBtOp(op, mac, 1)
	}
}

func doBtOp(op string, mac string, cnt int) {
	fmt.Println("~~", op, cnt, mac)
	cmd := exec.Command("bluetoothctl", op, mac)
	stdout, err := cmd.Output()
	fmt.Println("err", err, "stdout", string(stdout))
	if err != nil && cnt < 10 {
		time.Sleep(800 * time.Millisecond)
		doBtOp(op, mac, cnt+1)
	}
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

		if string(buf[:n]) == "bluebao-ping" {
			send()
			continue
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

func readLocal(confPath string) {
	if len(localEndpoints) != 0 {
		return
	}

	payload, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(err)
	}

	merge(payload)
}

func pingNw(justOnce bool) {
	for {
		fmt.Println("~~ pinging network for state")

		_, err := client.WriteTo([]byte("bluebao-ping"), clientAddr)
		if err != nil {
			fmt.Println("err writing", err)
		}

		time.Sleep(time.Second * 2)

		if justOnce || len(localEndpoints) != 0 {
			return
		}
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

func main() {
	var multicastAddr = flag.String("h", "192.168.0.255", "multicast address")
	var clientPort = flag.String("cp", "8830", "client port")
	var serverPort = flag.String("sp", "8829", "server port")

	var confPath = flag.String("c", "", "read config at path (optional if fetching from peers)")

	flag.Parse()

	localName, _ = os.Hostname()
	localEndpoints = make(map[string]endpoint)

	go startUI()
	time.Sleep(100 * time.Millisecond)
	startClient(*multicastAddr, *clientPort, *serverPort)
	go startServer(*serverPort)

	fmt.Printf("~~ bluebao starting, name %s\n\n", localName)
	if *confPath != "" {
		pingNw(true)
		readLocal(*confPath)
	} else {
		go pingNw(false)
	}

	select {}
}
