package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	udpPort            = 8285
	pathToScripts      = "/etc/overlay/setup_node"
	localPathToOverlay = "/etc/overlay/overlay_svc"
)

func createNode() {
	overlayName := promptOverlayName()
	overlayFile := getOverlayIPsLocalPath(overlayName)
	exists := doesOverlayExist(overlayFile)
	if !exists {
		fmt.Println("Overlay doesn't exist")
		os.Exit(1)
	}

	ipString, err := prompt("\nNode IP Address (IPv4)")
	if err != nil {
		fmt.Println("error taking input:", err)
	}
	ipBytes := []byte{}
	for _, ip := range strings.Split(ipString, ".") {
		bi, err := strconv.Atoi(ip)
		if err != nil {
			fmt.Println("Error parsing IP", err)
		}
		ipBytes = append(ipBytes, byte(bi))
	}
	if len(ipBytes) != 4 {
		fmt.Println("Invalid IPv4 IP")
		os.Exit(1)
	}

	user, err := prompt("\nNode username (user you log in as)")
	if err != nil {
		fmt.Println("error taking input:", err)
	}

	// parse overlay file
	overlayIPLookup := readOverlayFile(overlayFile)

	// don't 0 index -- subnet may be needed
	nodeNum := len(overlayIPLookup) + 1
	existingNodeNum := checkForNodeExisting(ipString, overlayIPLookup)
	if existingNodeNum >= 0 {
		nodeNum = existingNodeNum
		fmt.Println("Found existing node with that IP: node number", nodeNum)
		fmt.Println("Will overwrite")
	}

	// add new IP and node number
	overlayIPLookup[byte(nodeNum)] = &net.UDPAddr{IP: ipBytes, Port: udpPort}

	// save back to file
	writeOverlayFile(overlayFile, overlayIPLookup)

	// save user
	userFilePath := getUsersFilePath(overlayName)
	users := readUsersFile(userFilePath)
	users[nodeNum] = user
	writeUsersFile(userFilePath, users)

	createOverlaySystemdService(ipString, user, nodeNum)

	// need to deploy all overlays -- new ip in lookup
	deployOverlay(overlayName)
}

func command(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func createOverlaySystemdService(ip string, user string, nodeNum int) {
	err := command(pathToScripts+"create_overlay_svc.sh", user, ip, strconv.Itoa(nodeNum), remotePathToOverlayExecutable(), ipLookupRemotePath())
	checkErr(err, "error running create_overlay_svc")
}

func deployOverlayToNode(ip string, user string, overlayName string) {
	err := command(pathToScripts+"run_overlay.sh", user, ip, getOverlayIPsLocalPath(overlayName), ipLookupRemotePath(), remotePathToOverlayExecutable(), localPathToOverlay)
	checkErr(err, "error running run_overlay")
}

func readOverlayFile(ipLookupPath string) map[byte]*net.UDPAddr {
	var nodeIPLookup map[byte]*net.UDPAddr
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
	return nodeIPLookup
}

func checkForNodeExisting(ipString string, ipMap map[byte]*net.UDPAddr) int {
	for n, addr := range ipMap {
		if addr.IP.String() == ipString {
			return int(n)
		}
	}
	return -1
}

func writeOverlayFile(ipLookupPath string, ipLookup map[byte]*net.UDPAddr) {
	b, _ := json.Marshal(ipLookup)
	err := ioutil.WriteFile(ipLookupPath, b, 0644)
	if err != nil {
		fmt.Println("Unable to update IP Map")
		os.Exit(1)
	}
}

func writeUsersFile(userFilePath string, users map[int]string) {
	b, _ := json.Marshal(users)
	err := ioutil.WriteFile(userFilePath, b, 0644)
	if err != nil {
		fmt.Println("Unable to update users")
		os.Exit(1)
	}
}

func getUsersFilePath(overlayName string) string {
	return overlayDir + overlayName + "_users"
}

func readUsersFile(userFilePath string) map[int]string {
	var users map[int]string
	data, err := ioutil.ReadFile(userFilePath)
	if err != nil {
		fmt.Println("couldn't open user file:", err)
		os.Exit(1)
	}

	err = json.Unmarshal(data, &users)
	if err != nil {
		fmt.Println("couldn't read user file:", err)
		os.Exit(1)
	}
	return users
}

func checkErr(err error, msg string) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}
