package service

import (
	"github.com/gridsystem-back/internal/model"
	"log"
	"time"
)

var nodes = make(map[string]*model.Node)
var deviceMap = make(map[string]*model.DeviceInfo)

// AddNode 添加一个新的节点
func AddNode(node *model.Node) {
	nodes[node.ID] = node
}

// UpdateNodeStatus 更新节点状态
func UpdateNodeStatus(id, status string) {
	if node, exists := nodes[id]; exists {
		node.Status = status
		node.LastUpdated = time.Now().Format(time.RFC3339)
	}
}

// GetNodes 获取所有节点
func GetNodes() []*model.Node {
	var nodeList []*model.Node
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	return nodeList
}

// UpdateNodeWithDevices 更新节点状态并处理设备信息
func UpdateNodeWithDevices(id, status string, devices []model.DeviceInfo) {
	if node, exists := nodes[id]; exists {
		// 更新节点状态
		node.Status = status
		node.LastUpdated = time.Now().Format(time.RFC3339)
		// 更新设备信息
		node.Devices = devices

		log.Printf("Node %s updated with status: %s and devices: %+v", id, status, devices)
	} else {
		log.Printf("Node %s not found", id)
	}
	for _, d := range devices {
		deviceMap[d.Serial] = &d
	}
}
