package main

import (
	"github.com/gridsystem-node/config"
	"github.com/gridsystem-node/pkg/device/android"
	"github.com/gridsystem-node/pkg/heartbeat"
	"time"
)

// NodeStatus 用于表示节点的状态
type NodeStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	LastUpdated string `json:"last_updated"`
}

func init() {
	config.InitConfig()
	config.SetupLogger()
}

func main() {

	watchChan := make(chan string, 0)
	go android.WatchDevice(watchChan)

	heartQuitChan := make(chan string)
	heartbeat.InitHeartBeat()
	go heartbeat.SendHeartbeat(heartQuitChan)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		heartbeat.GenerateHeartBeat(android.GetDeviceList())
	}

}
