package main

import (
	"fmt"

	"golang.org/x/net/icmp"
)

func main() {
	icmp.ListenPacket("udp", "0.0.0.0:0")
	fmt.Println("Listening for ICMP packets")
}
