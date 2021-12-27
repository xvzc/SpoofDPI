package main

import (
    "net"
    "log"
    "SpoofDPI/handler"
)

func main() {
	log.Println("##### Listening 8080..")

    listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	for {
		connClient, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}

		log.Println("##### New connection", connClient.RemoteAddr())

        go handler.HandleClientRequest(connClient)
	}
}

