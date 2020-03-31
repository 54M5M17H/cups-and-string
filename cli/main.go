package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printAllHelp()
		return
	}

	switch os.Args[1] {
	case "overlay":
		if len(os.Args) < 3 {
			printOverlayHelp()
			return
		}

		switch os.Args[2] {
		case "create":
			createNewOverlay()
		case "list":
			listOverlays()
		case "redeploy":
			deployOverlay("")
		default:
			printOverlayHelp()
		}

	case "node":
		if len(os.Args) < 3 {
			printNodeHelp()
			return
		}

		switch os.Args[2] {
		case "create":
			createNode()
		case "list":
			fmt.Println("List nodes")
		default:
			printNodeHelp()
		}

	default:
		printAllHelp()
	}
}
