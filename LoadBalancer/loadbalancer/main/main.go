package main

import (
	"fmt"
	"io"
	"net"
)

var BackendServers = []string{
	"localhost:8081", //the port can be same but for local single machine test it should be different
	"localhost:8082",
}

var CurrentServerIndex = 0

// This approach is inefficient because it performs blocking I/O.
// The load balancer waits for the server's response before proceeding,
// preventing it from accepting new requests or concurrently forwarding data to the client.

func ForwardRequestWithoutThread(clientRequest net.Conn) error {
	fmt.Println("Forwarding request to server", CurrentServerIndex)
	//1. select the current server
	currentServerUrl := BackendServers[CurrentServerIndex]

	connectToServer, err := net.Dial("tcp", currentServerUrl)
	if err != nil {
		return err
	}

	defer connectToServer.Close()

	// To move to the next server after the request is served
	CurrentServerIndex = (CurrentServerIndex + 1) % len(BackendServers)

	//2. copy the response from the backend to the client
	// io.copy is used to copy the response from the backend to the client vice versa and it take args dest and src
	io.Copy(connectToServer, clientRequest)

	//3. copy the response from the client to the backend
	io.Copy(clientRequest, connectToServer)

	return nil

}

func main() {
	//start a loa balance server
	listner, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listner.Close()

	fmt.Println("load balancer started")

	// this the incomming / client connection accepter
	incommingRequests, err := listner.Accept()

	if err != nil {
		panic(err)
	}

	err = ForwardRequestWithoutThread(incommingRequests)
	if err != nil {
		fmt.Println("error forwarding request", err)
	}
	fmt.Println("request forwarded to server main close")

}
