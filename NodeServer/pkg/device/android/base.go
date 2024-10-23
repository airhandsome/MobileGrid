package android

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

var webStream net.Conn
var forwardStream net.Conn
var listen net.Listener

type TcpProxy struct {
	Serial     string
	LocalPort  int
	RemoteHost string
	RemotePort int
}

func (t *TcpProxy) StopProxy() {
	if listen != nil {
		listen.Close()
		listen = nil
	}

	if webStream != nil {
		webStream.Close()
		webStream = nil
	}
	if forwardStream != nil {
		forwardStream.Close()
		forwardStream = nil
	}
}

func (t *TcpProxy) StartProxy() {
	var err error
	listen, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", t.LocalPort))

	if err != nil {
		log.Println("listen failed, err: ", err)
		return
	}

	go func() {
		for listen != nil {
			conn, err := listen.Accept()
			if err != nil {
				log.Println("connect failed, err: ", err)
				continue
			}
			go t.HandleWebStream(conn)
			time.Sleep(time.Second)
		}
	}()
}

func (t *TcpProxy) HandleWebStream(conn net.Conn) {
	go t.ForwardStream()
	//make sure forward socket is established
	time.Sleep(500 * time.Millisecond)
	webStream = conn
	for webStream != nil {
		reader := bufio.NewReader(conn)
		var buf [1024]byte
		n, err := reader.Read(buf[:])
		if err != nil {
			log.Println("Got error message from web: ", err)
			break
		}
		if forwardStream != nil {
			forwardStream.Write(buf[:n])
		}
	}
}

func (t *TcpProxy) ForwardStream() error {

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", t.RemotePort))
	if err != nil {
		log.Printf("Can't connect to screen port %d \n", t.RemotePort)
		return err
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 60))
	forwardStream = conn
	for forwardStream != nil {
		chunk := make([]byte, 10240)
		cnt, err := conn.Read(chunk)
		if err != nil {
			log.Printf("[%s]Got error from forward stream: %s\n", t.Serial, err.Error())
			if _, ok := err.(*net.OpError); ok {
				return err
			}
			break
		}
		chunk = chunk[:cnt]
		if webStream != nil {
			webStream.Write(chunk)
		}
	}
	return nil
}
