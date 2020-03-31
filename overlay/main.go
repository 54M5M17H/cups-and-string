package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

const (
	interfaceName = "tun0"
	mtu           = 1300
	udpSize       = 1024
	udpPort       = 8285
)

var (
	ipLookupPath = os.Getenv("NODE_IP_LOOKUP_PATH")
	nodeNum      = os.Getenv("NODE_NUM")
)

func calcTunSubnet() string {
	return fmt.Sprintf("10.10.%s.0/16", nodeNum) // ensure each node's source tun IP is unique
}

func main() {
	generateIPMap()
	tunSubnet := calcTunSubnet()
	ensureDocker()

	// open a new tun
	cfg := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:        interfaceName,
			Persist:     false, // ?????
			Permissions: nil,
			MultiQueue:  false, // allows parallelisation -- not for now
		},
	}

	tun, err := water.New(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// io.ReadWriteCloser
	// Reader
	// Writer
	// Closer

	defer tun.Close()

	err = exec.Command("sudo", "ip", "link", "set", "dev", interfaceName, "mtu", strconv.Itoa(mtu)).Run()
	if err != nil {
		fmt.Println("Error setting mtu", err)
		return
	}

	err = exec.Command("sudo", "ip", "addr", "add", tunSubnet, "dev", interfaceName).Run()
	if err != nil {
		fmt.Println("Error setting subnet", err)
		return
	}

	err = exec.Command("sudo", "ip", "link", "set", "dev", interfaceName, "up").Run()
	if err != nil {
		fmt.Println("Error turning on interface", err)
		return
	}

	err = exec.Command("sudo", "ip", "route", "add", "10.10.0.0/16", "dev", interfaceName).Run()
	if err != nil {
		fmt.Println("Error creating tun route", err)
		if err.Error() == "exit status 2" {
			fmt.Println("Route already exists")
		} else {
			return
		}
	}

	go read(tun)
	go listener(tun)

	// listen for interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	sig := <-c
	fmt.Println("Got signal", sig)
	tun.Close()
	fmt.Println("Closed tun")

}

func read(intf io.Reader) {
	fmt.Println("Listening.......")
	i := 0
	for {
		b := make([]byte, mtu)
		n, err := intf.Read(b)
		if err != nil {
			fmt.Println("Error reading: ", err)
			return
		}
		if n <= 0 {
			// time.Sleep(time.Second * 1)
			i++
			if (i % 100) == 0 {
				fmt.Println("Still listening")
				i = 0
			}
			continue
		}

		if waterutil.IsIPv6(b) {
			fmt.Println("Ignoring IPv6 packet -- don't care about these atm")
			continue
		}

		fmt.Printf("Read %d bytes \n", n)
		printIPPacketDetails(b)
		nodeAddr := getNodeAddr(b)
		if nodeAddr == nil {
			fmt.Println("No match for IP", waterutil.IPv4Destination(b))
			continue
		}
		fmt.Println(b[:n])

		fmt.Println("Sharing with other node --", nodeAddr)
		udpClient(*nodeAddr, b[:n])
	}
}

func calcDockerSubnet() (string, int) {
	subnetSize := 24
	return fmt.Sprintf("10.10.%s.0/%d", nodeNum, subnetSize), subnetSize
}

func ensureDocker() {
	o, err := exec.Command("docker", "network", "inspect", "-f='{{ (index .IPAM.Config 0).Subnet }}'", "bridge").Output()
	if err != nil {
		fmt.Println("Unable to check docker", err)
		os.Exit(1)
	}

	// [1:len(o) -2] -- remove quotes
	currentSubnet := string(o[1 : len(o)-2])
	expectedSubnet, subnetSize := calcDockerSubnet()
	if currentSubnet == expectedSubnet {
		fmt.Println("Ensured docker subnet")
		return
	}

	dockerConfigFile := fmt.Sprintf(`{ "default-address-pools": [ {"base":"%s/%d","size":%d} ] }`, currentSubnet, subnetSize, subnetSize)
	ioutil.WriteFile("/etc/docker/daemon.json", []byte(dockerConfigFile), 0644)
	err = exec.Command("systemctl", "restart", "docker").Run()
	if err != nil {
		fmt.Println("error restarting docker", err)
		os.Exit(1)
	}
	fmt.Println("Configured Docker")
}

func printIPPacketDetails(b []byte) {
	fmt.Println("Destination IP", waterutil.IPv4Destination(b))
	fmt.Println("Destination Port", waterutil.IPv4DestinationPort(b))
	fmt.Println("Source IP", waterutil.IPv4Source(b))
	fmt.Println("Source Port", waterutil.IPv4SourcePort(b))
}

func udpClient(addr net.UDPAddr, payload []byte) {
	var localAddr net.UDPAddr
	conn, err := net.DialUDP("udp", &localAddr, &addr)
	if err != nil {
		fmt.Println("Error dialing udp", err)
		os.Exit(1)
	}
	defer conn.Close()
	conn.Write(payload)
}

func listener(tun io.Writer) {
	listenAddr, err := net.ResolveUDPAddr("udp", ":8285")
	if err != nil {
		fmt.Println("Error creating UDP listen address")
		os.Exit(1)
	}

	server, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		fmt.Println("Error starting UDP server", err)
		os.Exit(1)
	}
	fmt.Println("Listening for UDP packets on :8285")

	defer server.Close()

	for {
		b := make([]byte, udpSize)
		n, addr, err := server.ReadFromUDP(b)
		if err != nil {
			fmt.Println("Error reading UDP packet", err)
			os.Exit(1)
		}
		if n <= 0 {
			continue
		}

		fmt.Println("Inbound packet", addr)
		fmt.Printf("Packet has %d bytes \n", n)
		printIPPacketDetails(b)
		fmt.Println("Writing to network stack...")
		nW, err := tun.Write(b[:n])
		if err != nil {
			fmt.Println("Error writing to stack", err)
			os.Exit(1)
		}
		fmt.Printf("Written %d bytes to stack \n", nW)
		fmt.Println(b[:n])
	}
}

// var (
// 	nodeLookup = map[byte]*net.UDPAddr{
// 		byte(1): &net.UDPAddr{IP: []byte{192, 168, 0, 60}, Port: udpPort, Zone: ""},
// 		byte(2): &net.UDPAddr{IP: []byte{192, 168, 0, 61}, Port: udpPort, Zone: ""},
// 	}
// )

var nodeIPLookup map[byte]*net.UDPAddr

// getNodeAddr extracts the node number from the container IP and looks up the node UDP address
func getNodeAddr(ipPacket []byte) *net.UDPAddr {
	// containerIPs take the form: 10.20.NODE_NUMBER.XXX
	// we want to extract that NODE_NUMBER
	// then we can use it to lookup the node IP

	containerIP := waterutil.IPv4Destination(ipPacket)
	// containerIP is a 16-byte array -- length of a IPv6 packet
	// in this case the first 12-bytes are 0
	nodeMaskByte := containerIP[14] // second to last byte
	nodeIP := nodeIPLookup[nodeMaskByte]
	return nodeIP
}

func generateIPMap() {
	data, err := ioutil.ReadFile(ipLookupPath)
	if err != nil {
		fmt.Println("couldn't open IP Lookup:", err)
		os.Exit(1)
	}

	err = json.Unmarshal(data, &nodeIPLookup)
	if err != nil {
		fmt.Println("couldn't read IP Lookup:", err)
		os.Exit(1)
	}
	fmt.Println(nodeIPLookup)
}
