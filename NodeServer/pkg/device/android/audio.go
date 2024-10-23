package android

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"time"
)

/*
*
Testing method:

	1.for video stream: refresh, change miniprogram and return, change resolution
	2. for control stream: touch, move, scroll and input keys
*/

type AudioProxy struct {
	screenAudioStream *websocket.Conn
	audioQuitChan     chan int // quit for audio
	audioData         Audio
	AdbClient         AdbConnection
	TcpProxy
}

type Audio struct {
	frameBody []byte
	length    int
	LeftOver  []byte
}

func (p *AudioProxy) StopProxy() {
	log.Printf("[%s]Send stop scrcpy proxy.\n", p.AdbClient.abstractSerial())
	if p.audioQuitChan != nil {
		p.audioQuitChan <- 1
	}
}

func (p *VideoProxy) audioProxy(quit chan int) {
	defer func() {
		if p.audioStream != nil {
			p.audioStream.Close()
			p.audioStream = nil
		}
	}()

	r := gin.Default()
	r.GET("/", p.HandleWebAudioStream)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", p.AudioProxy.LocalPort),
		Handler: r,
	}
	go func() {
		log.Printf("[%s]audio web server started\n", p.AdbClient.abstractSerial())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-quit
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("[%s]Server Shutdown %s\n", p.AdbClient.abstractSerial(), err.Error())
	}
	log.Printf("[%s]Websocket Server Shutdown\n", p.AdbClient.abstractSerial())
}
func (p *VideoProxy) HandleWebAudioStream(c *gin.Context) {
	if p.screenAudioStream != nil {
		p.screenAudioStream.Close()
		p.screenAudioStream = nil
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	p.screenAudioStream = ws

	for p.screenAudioStream != nil {
		time.Sleep(time.Second)
	}
}

func (p *VideoProxy) HandleAudioStream(quit chan int) {
	for p.audioStream != nil {
		select {
		case <-quit:
			log.Println("Quit device video stream")
			return
		default:
			if !p.DeviceAudioParse() {
				log.Println("parse device audio stream error")
			}
		}
	}
}
func byteArrayToInt(bytes []byte) int {
	var intValue int
	for i, b := range bytes {
		intValue |= int(b) << (8 * (len(bytes) - 1 - i))
	}
	return intValue
}
func (p *VideoProxy) DeviceAudioParse() bool {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s]:Audio frame parse error %s\n", p.AdbClient.abstractSerial(), err)
			return
		}
	}()
	if p.audioStream == nil {
		return false
	}
	chunk := make([]byte, 4)
	_, err := p.audioStream.Read(chunk)
	if err != nil && err != io.EOF {
		log.Println("Got Error when receive from device: ", err)
		log.Printf("get message %s", string(chunk))
		return true
	}

	length := byteArrayToInt(chunk)
	// 4 byte ASCII OPUS
	// 8 byte TS + 4 byte length + data(length)
	//log.Println("audioCodec audio length: " + strconv.Itoa(length))
	chunk = make([]byte, length)
	_, err = p.audioStream.Read(chunk)
	if err != nil {
		if err != io.EOF {
			log.Println("Got Error when receive from device: ", err)
			return false
		}
		return true
	}

	if p.screenAudioStream != nil {
		p.screenAudioStream.WriteMessage(2, chunk)
	}

	return true
}
