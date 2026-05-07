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

func ForwardRequest(clientRequest net.Conn) error {
	//1. select the current server
	currentServerUrl := BackendServers[CurrentServerIndex]

	connectToServer, err := net.Dial("tcp", currentServerUrl)
	if err != nil {
		return err
	}

	defer connectToServer.Close()

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

	err = ForwardRequest(incommingRequests)
	if err != nil {
		fmt.Println("error forwarding request", err)
	}

}
