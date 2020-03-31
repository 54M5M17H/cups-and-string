package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"
)

const (
	overlayDir = "/etc/overlay/records/"
)

func prompt(msg string) (string, error) {
	var input string
	fmt.Printf("%v: ", msg)
	_, e := fmt.Scanf("%s", &input)
	return input, e
}

func promptOverlayName() string {
	r := regexp.MustCompile("[^a-z]")
	overlayName, _ := prompt("\nOverlay name - lowercase characters only")
	if r.MatchString(overlayName) {
		fmt.Println("Name can only contain lower case letters")
		return promptOverlayName()
	}
	return overlayName
}

func createNewOverlay() {
	overlayName := promptOverlayName()

	err := os.MkdirAll(overlayDir, os.ModePerm)
	if err != nil {
		fmt.Println("Couldn't make dir:", err)
		os.Exit(1)
	}

	overlayFile := getOverlayIPsLocalPath(overlayName)
	exists := doesOverlayExist(overlayFile)

	if exists {
		fmt.Println("An overlay already exists with the name", overlayName)
		os.Exit(1)
	}

	writeOverlayFile(overlayFile, map[byte]*net.UDPAddr{})

	userFilePath := getUsersFilePath(overlayName)
	writeUsersFile(userFilePath, map[int]string{})

	fmt.Println("Overlay created")
}

func getOverlayIPsLocalPath(overlayName string) string {
	return overlayDir + overlayName
}

func doesOverlayExist(overlayFile string) bool {
	exists, err := os.Stat(overlayFile)
	if err != nil {
		return false
	}
	if exists != nil {
		return true
	}

	return false
}

func listOverlays() {
	files, err := ioutil.ReadDir(overlayDir)
	if err != nil {
		fmt.Println("Error reading overlays dir:", err)
	}
	for _, f := range files {
		if strings.Contains(f.Name(), "_users") {
			continue
		}
		fmt.Println(f.Name())
	}
}

func deployOverlay(overlayName string) {
	if overlayName == "" {
		overlayName = promptOverlayName()
	}
	overlayFile := getOverlayIPsLocalPath(overlayName)
	exists := doesOverlayExist(overlayFile)
	if !exists {
		fmt.Println("That overlay does not exist")
		os.Exit(1)
	}

	overlayData := readOverlayFile(overlayFile)
	userFilePath := getUsersFilePath(overlayName)
	nodeUsers := readUsersFile(userFilePath)

	for n, addr := range overlayData {
		nodeNum := int(n)
		user := nodeUsers[nodeNum]
		deployOverlayToNode(addr.IP.String(), user, overlayName)
	}
}
