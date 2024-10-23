package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/gridsystem-back/internal/model"
	"github.com/gridsystem-back/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/nodes", GetNodes)
	r.POST("/node", AddNode)
	r.PUT("/node/:id/status", UpdateNodeStatus)
	r.GET("/node/:id/ws", GetNodeStatus)
}

func GetNodes(c *gin.Context) {
	nodes := service.GetNodes()
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

func AddNode(c *gin.Context) {
	var node model.Node
	if err := c.ShouldBindJSON(&node); err == nil {
		service.AddNode(&node)
		c.JSON(http.StatusCreated, gin.H{"message": "Node added successfully"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func UpdateNodeStatus(c *gin.Context) {
	id := c.Param("id")
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err == nil {
		if status, ok := data["status"]; ok {
			service.UpdateNodeStatus(id, status)
			c.JSON(http.StatusOK, gin.H{"message": "Node status updated"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func GetNodeStatus(c *gin.Context) {
	id := c.Param("id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var heartbeat model.Node
		fmt.Println(string(message))
		if err := json.Unmarshal(message, &heartbeat); err != nil {
			log.Println("Unmarshal error:", err)
			continue
		}

		// 更新节点状态
		service.UpdateNodeWithDevices(id, heartbeat.Status, heartbeat.Devices)

		log.Printf("Received heartbeat: %+v\n", heartbeat)
	}
}
