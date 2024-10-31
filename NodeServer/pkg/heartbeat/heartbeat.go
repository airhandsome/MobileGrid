package heartbeat

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/gridsystem-node/config"
	"log"
	"net/http"
	"net/url"
	"time"
)

// NodeStatus 用于表示节点的状态
type NodeStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	LastUpdated string       `json:"last_updated"`
	Devices     []DeviceInfo `json:"devices"`
}

type DeviceInfo struct {
	Serial   string `json:"serial"`
	Model    string `json:"model"`
	State    string `json:"state"`
	Product  string `json:"product"`
	Platform string `json:"platform"`
}

var heartBeatChan chan string
var ws *websocket.Conn

func InitHeartBeat() {
	heartBeatChan = make(chan string, 10)
	var err error
	var resp *http.Response
	ws, resp, err = createWebSocketClient(config.WebsocketURL + "/node/" + config.NodeId + "/ws")
	if err != nil {
		log.Fatalf("Failed to create WebSocket client: %v", err)
	}
	log.Printf("WebSocket connection established with resp %v\n", resp)
}
func createWebSocketClient(wsURL string) (*websocket.Conn, *http.Response, error) {
	// 解析 WebSocket URL
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, nil, err
	}

	// 创建 WebSocket 拨号器
	dialer := websocket.DefaultDialer

	// 设置拨号超时时间
	dialer.HandshakeTimeout = 5 * time.Second

	// 拨号连接到 WebSocket 服务器
	conn, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, resp, err
	}

	return conn, resp, nil
}
func SendHeartbeat(quitChan <-chan string) {
	for {
		select {
		case status := <-heartBeatChan:
			if ws == nil {
				log.Println("WebSocket connection is nil, skipping heartbeat")
				continue
			}
			err := ws.WriteMessage(websocket.TextMessage, []byte(status))
			if err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
			} else {
				log.Println("Heartbeat sent successfully")
			}
		case <-quitChan:
			log.Println("Received quit signal, exiting sendHeartbeat loop")
			return
		}
	}
}

func GenerateHeartBeat(devices []DeviceInfo) {
	status := NodeStatus{
		ID:          config.NodeId,
		Name:        config.NodeName,
		Status:      "online",
		LastUpdated: time.Now().Format(time.RFC3339),
		Devices:     devices,
	}

	jsonBody, err := json.Marshal(status)
	if err != nil {
		log.Println("Failed to marshal heartbeat")
	}
	heartBeatChan <- string(jsonBody)

}
