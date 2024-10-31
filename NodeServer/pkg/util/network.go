package util

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

var portLock sync.Mutex
var portStart = 15000
var portEnd = 65535
var currentPort = 15000

func GetFreePort() (int, error) {
	portLock.Lock()
	defer portLock.Unlock()

	for i := 0; i < 10; i++ {
		currentPort = (currentPort+1)%(portEnd-portStart+1) + portStart
		if !isPortInUse(currentPort) {
			return currentPort, nil
		}
	}

	return 0, fmt.Errorf("cannot find a free port in the specified range")
}
func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)), time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// checkPort 检查指定端口是否开启
func checkPort(port string) bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// MonitorPort 每隔1秒检测指定端口是否开启
func MonitorPort(port string) {
	for {
		if checkPort(port) {
			fmt.Println("Port", port, "is open.")
			break
		}
		fmt.Println("Port", port, "is closed. Retrying in 1 second...")
		time.Sleep(1 * time.Second)
	}
}
