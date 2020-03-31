package main

import "fmt"

func dockerSubnet(nodeNum int) (string, string) {
	return fmt.Sprintf("10.10.%d.0", nodeNum), "24"
}

func ipLookupRemotePath() string {
	return "/tmp/overlay_ip_lookup"
}

func remotePathToOverlayExecutable() string {
	return "/tmp/overlay/overlay_bin"
}

func tunSubnet(nodeNum int) (string, string) {
	return fmt.Sprintf("10.10.%d.0", nodeNum), "16"
}
