package main

import (
	"fmt"
	"net"
)

func main() {
	// log.Fatal(agent.StartDNS())
	a := "172.17.0.1:5367"
	/* Lets prepare a address at any address at port 10001*/
	addr, err := net.ResolveUDPAddr("udp", a)
	if err != nil {
		panic(err)
	}
	fmt.Println("listening on", a)
	/* Now listen at selected port */
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}
		fmt.Printf("received: %s from: %s\n", string(buf[0:n]), addr)

		if err != nil {
			fmt.Println("error: ", err)
		}

		_, _ = conn.WriteTo(buf[0:n], addr)
	}
	// }
}
