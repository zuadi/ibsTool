//go:build discovery

package main

import (
	"fmt"
	"ibsTool/udpDiscovery"
)

func main() {
	fmt.Println(1)
	fmt.Println(udpDiscovery.DiscoverDevices())
}
