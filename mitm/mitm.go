package mitm

import (
    "net"
    "log"
    "io/ioutil"
    "fmt"
)

func GoGoSing(clientConn net.Conn, remoteConn net.Conn, data []byte) {
    _, write_err := remoteConn.Write(data)
    if write_err != nil {
        log.Fatal("failed:", write_err)
        return
    }
    defer remoteConn.(*net.TCPConn).CloseWrite()

    // Read from the server
    buf, read_err := ioutil.ReadAll(remoteConn)
    if read_err != nil {
        log.Fatal("failed:", read_err)
        return
    }

    fmt.Println()
    log.Println()
    fmt.Println("##### Response from the server: ")
    fmt.Println(string(buf))


    // Write to client
    _, write_err = clientConn.Write(buf)
    if write_err != nil {
        log.Fatal("failed:", write_err)
        return
    }
    defer clientConn.(*net.TCPConn).CloseWrite()
}
