package main

import (
	"fmt"
	"net"
	"time"

	"github.com/docker/libchan/bus"
)

func client1(client1Conn net.Conn) {
	// USAGE client1
	client, err := bus.NewNetClient(client1Conn, "client-1") //client1Conn inherited or passed from Bus
	if err != nil {
		panic(err)
	}
	recv, err := client.Register("test-address-1")
	if err != nil {
		panic(err)
	}

	var v map[string]interface{}
	err = recv.Receive(&v) // error ignored
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", v)
	// Usage client1 END
}

func client2(client2Conn net.Conn) {
	time.Sleep(5 * time.Millisecond)
	// USAGE client2
	client, err := bus.NewNetClient(client2Conn, "client-2") //client2Conn inherited or passed from Bus
	if err != nil {
		panic(err)
	}

	err = client.Message("test-address-1", map[string]interface{}{"key": "any libchan value"}) // error ignored
	if err != nil {
		panic(err)
	}
	// USAGE client2 END
}

func main() {
	// USAGE bus
	b := bus.NewBus()
	client1Conn, bus1 := net.Pipe()
	client2Conn, bus2 := net.Pipe()

	err := b.Connect(bus1) // error ignored
	if err != nil {
		panic(err)
	}
	err = b.Connect(bus2) // error ignored
	if err != nil {
		panic(err)
	}
	// USAGE bus END

	go client2(client2Conn)
	client1(client1Conn)
}
