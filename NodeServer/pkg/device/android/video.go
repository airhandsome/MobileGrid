package android

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/electricbubble/gadb"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/gridsystem-node/config"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Banner struct {
	version       int
	length        int
	pid           uint32
	realWidth     uint32
	realHeight    uint32
	virtualWidth  uint32
	virtualHeight uint32
	orientation   int
	quirks        int
}

type VideoProxy struct {
	videoStream         net.Conn
	controlStream       net.Conn
	audioStream         net.Conn
	screenStream        *websocket.Conn
	screenAudioStream   *websocket.Conn
	ControlPort         int
	UploadVideo         bool
	AdbClient           AdbConnection
	quitChan            chan int //quit for ws listen
	deviceQuitChan      chan int //quit for video with phone
	deviceAudioQuitChan chan int // quit for audio with phone
	coldSignalChan      chan int // quit for cold start

	imageData Image
	videoData Video
	mutex     sync.Mutex
	Speed
	TcpProxy
	AudioProxy
}

type Speed struct {
	showSpeed   bool
	serverBytes float64
}

type AdbConnection struct {
	Device         *gadb.Device
	AbstractSerial string
	Cmd            string
	process        *exec.Cmd
}

type Video struct {
	videoMode   bool
	hasBanner   bool
	hasRotation bool
	banner      *Banner
	frameBody   []byte
	lackLength  int
}

type Image struct {
	frameBody []byte
	length    int
	leftOver  []byte
}

var originalCmd string

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
	return true
}}

func (adb *AdbConnection) StartVideo() {
	adb.process = exec.Command(adbPath, "-s", adb.Device.Serial(), "shell", adb.Cmd)
	//for debug
	adb.process.Stdout = os.Stdout
	if config.LogFile != nil {
		adb.process.Stderr = config.LogFile
	} else {
		adb.process.Stderr = os.Stdout
	}

	err := adb.process.Start()
	if err != nil {
		log.Printf("[%s]:Device adb exit for %s\n", adb.abstractSerial(), err.Error())
	}
}

func (adb *AdbConnection) abstractSerial() string {
	return adb.AbstractSerial
}

func (adb *AdbConnection) StopProxy() {
	if adb.process != nil {
		adb.process.Process.Kill()
	}
	adb.process = nil
}

func (v *Video) ResetParam() {
	v.hasBanner = false
	v.hasRotation = false
	v.frameBody = nil
}

func (i *Image) ResetParam() {
	i.frameBody = []byte{}
	i.length = 0
}

func (p *VideoProxy) StopProxy() {
	log.Printf("[%s]Send stop scrcpy proxy.\n", p.AdbClient.abstractSerial())
	if p.quitChan != nil {
		p.quitChan <- 1
	}
	if p.coldSignalChan != nil {
		p.coldSignalChan <- 1
	}
	//quit audio connection with phone
	if p.deviceAudioQuitChan != nil {
		p.deviceAudioQuitChan <- 1
	}

	//quit video connection with phone
	if p.deviceQuitChan != nil {
		p.deviceQuitChan <- 1
	}
}

func (p *VideoProxy) HandleWebStream(c *gin.Context) {

	//close old websocket when handle new connection
	if p.screenStream != nil {
		p.screenStream.Close()
		p.screenStream = nil
	}

	//upgrade http to websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	p.screenStream = ws

	// handle video stream
	if p.videoStream == nil {
		log.Println("video proxy is not ready, please try again later")
		go p.Connect()
	}

	log.Printf("[%s]: get connection from web\n", p.AdbClient.abstractSerial())
	for p.screenStream != nil {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			if p.screenStream != nil {
				p.screenStream.Close()
				p.screenStream = nil
			}
			return
		}

		datas := map[string]interface{}{}
		err = json.Unmarshal(msg, &datas)
		if err != nil {
			return
		}
		if msgType, ok := datas["msg_type"]; ok {
			switch int(msgType.(float64)) {
			case CONTROL_MSG_TYPE_INJECT_KEYCODE:
				buf := make([]byte, 10)
				buf[0] = float2byte(datas["msg_type"].(float64))
				buf[1] = float2byte(datas["msg_inject_keycode_action"].(float64))
				binary.BigEndian.PutUint32(buf[2:6], uint32(KeyCodeToAndroid[datas["msg_inject_keycode_keycode"].(string)]))
				binary.BigEndian.PutUint32(buf[6:10], uint32(datas["msg_inject_keycode_metastate"].(float64)))
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case CONTROL_MSG_TYPE_INJECT_TEXT:
				data := datas["message"].(string)
				txt := []byte(data)
				buf := make([]byte, 3)
				buf[0] = float2byte(datas["msg_type"].(float64))
				binary.BigEndian.PutUint16(buf[1:3], uint16(len(txt)))
				buf = append(buf, txt...)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT:
				touch := datas["msg_inject_touch_position"].(map[string]interface{})
				touchWidth, touchHeight := p.videoData.banner.realWidth, p.videoData.banner.realHeight
				if (touch["width"].(float64) > touch["height"].(float64)) != (touchWidth > touchHeight) {
					touchWidth, touchHeight = touchHeight, touchWidth
				}
				buf := make([]byte, 26)
				buf[0] = byte(CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT)
				buf[1] = float2byte(datas["msg_inject_touch_action"].(float64))

				binary.BigEndian.PutUint32(buf[2:6], uint32(datas["msg_inject_touch_index"].(float64)))
				binary.BigEndian.PutUint32(buf[6:10], uint32(touch["x"].(float64)*float64(touchWidth)))
				binary.BigEndian.PutUint32(buf[10:14], uint32(touch["y"].(float64)*float64(touchHeight)))
				binary.BigEndian.PutUint16(buf[14:16], uint16(touchWidth))
				binary.BigEndian.PutUint16(buf[16:18], uint16(touchHeight))
				binary.BigEndian.PutUint32(buf[18:22], uint32(1)) //touch pressure default 1
				binary.BigEndian.PutUint32(buf[22:26], uint32(1)) // touch buttons default 1
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT:
				scroll := datas["msg_inject_scroll_position"].(map[string]interface{})
				touchWidth, touchHeight := p.videoData.banner.realWidth, p.videoData.banner.realHeight
				if (scroll["width"].(float64) > scroll["height"].(float64)) != (touchWidth > touchHeight) {
					touchWidth, touchHeight = touchHeight, touchWidth
				}
				buf := make([]byte, 21)
				buf[0] = byte(CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT)
				binary.BigEndian.PutUint32(buf[1:5], uint32(scroll["x"].(float64)*float64(touchWidth)))
				binary.BigEndian.PutUint32(buf[5:9], uint32(scroll["y"].(float64)*float64(touchHeight)))
				binary.BigEndian.PutUint16(buf[9:11], uint16(touchWidth))
				binary.BigEndian.PutUint16(buf[11:13], uint16(touchHeight))
				binary.BigEndian.PutUint32(buf[13:17], uint32(datas["msg_inject_scroll_horizontal"].(float64)))
				binary.BigEndian.PutUint32(buf[17:], uint32(datas["msg_inject_scroll_vertical"].(float64)))
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}
			case CONTROL_MSG_TYPE_BACK_OR_SCREEN_ON:
				buf := make([]byte, 1)
				buf[0] = byte(CONTROL_MSG_TYPE_BACK_OR_SCREEN_ON)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}
			case CONTROL_MIDIFY_IMAGE_SETTING:
				//buf := make([]byte, 13)
				if p.AdbClient.process != nil {
					p.AdbClient.process.Process.Kill()
					p.AdbClient.process = nil
				}

				time.Sleep(1 * time.Second)
				p.AdbClient.Cmd = originalCmd + fmt.Sprintf(" -r %d -b %d -P %d -Q %d", int(datas["maxFps"].(float64)), int(datas["bitrate"].(float64)), int(datas["scale"].(float64)), int(datas["quality"].(float64)))
				log.Printf("[%s]: %s\n", p.AdbClient.abstractSerial(), p.AdbClient.Cmd)
				go p.AdbClient.StartVideo()
				if p.videoStream != nil {
					p.videoStream.Close()
					p.videoStream = nil
				}
				if p.controlStream != nil {
					p.controlStream.Close()
					p.controlStream = nil
				}
				if p.audioStream != nil {
					p.audioStream.Close()
					p.audioStream = nil
				}
				time.Sleep(2 * time.Second)
				p.Connect()

			case CONTROL_MSG_TYPE_ROTATE_DEVICE:
				log.Printf("[%s]: received rotate cmd\n", p.AdbClient.abstractSerial())
			case CONTROL_MSG_TYPE_RESET_VIDEO_STREAM:
				log.Printf("[%s]set video mode\n", p.AdbClient.abstractSerial())
				p.videoData.hasBanner = false
				p.videoData.videoMode = true
				//p.videoData.ResetParam()
				buf := make([]byte, 1)
				buf[0] = byte(CONTROL_MSG_TYPE_RESET_VIDEO_STREAM)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case CONTROL_MSG_TYPE_RESET_IMAGE_STREAM:
				log.Printf("[%s]: set image mode\n", p.AdbClient.abstractSerial())
				p.videoData.videoMode = false
				p.videoData.hasRotation = false
				p.videoData.frameBody = nil

				buf := make([]byte, 1)
				buf[0] = byte(CONTROL_MSG_TYPE_RESET_IMAGE_STREAM)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}
				p.imageData.ResetParam()

			case CONTROL_MSG_TYPE_MINIPROGRAM:
				log.Printf("[%s]: set image mode to go to miniprogram\n", p.AdbClient.abstractSerial())
				if !p.videoData.videoMode {
					continue
				}
				p.AdbClient.process.Process.Kill()
				p.videoData.ResetParam()
				p.imageData.ResetParam()
				p.videoStream = nil
				p.controlStream = nil
				p.audioStream = nil
				time.Sleep(1 * time.Second)
				go p.AdbClient.StartVideo()
				//p.Connect()

			case SCREEN_ORIENTATION_LANDSCAPE:
				buf := make([]byte, 1)
				buf[0] = byte(SCREEN_ORIENTATION_LANDSCAPE)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case SCREEN_ORIENTATION_PORTRAIT:
				buf := make([]byte, 1)
				buf[0] = byte(SCREEN_ORIENTATION_PORTRAIT)
				if p.controlStream != nil {
					p.controlStream.Write(buf)
				}

			case RECORD_SERVER_SPEED:
				if p.Speed.showSpeed == datas["msg_value"].(bool) {
					break
				}
				p.Speed.showSpeed = datas["msg_value"].(bool)
				if p.Speed.showSpeed {
					go func() {
						for p.Speed.showSpeed {
							speed := p.serverBytes / 1024 / 3 //KB/S  or Byte / ms
							p.serverBytes = 0
							if p.screenStream != nil {
								p.mutex.Lock()
								p.screenStream.WriteJSON(map[string]string{"speed": fmt.Sprintf("%.3f", speed)})
								p.mutex.Unlock()
							}
							time.Sleep(3 * time.Second)
						}
						log.Println("exit show speed mode")
					}()
				}

			}
		}
	}
	log.Println("exit handle web stream")
}
func (p *VideoProxy) videoProxy(quit chan int) {
	defer func() {
		log.Println("enter into video proxy")
		if p.AdbClient.process != nil {
			p.AdbClient.process.Process.Kill()
			p.AdbClient.process = nil
		}
		if p.controlStream != nil {
			p.controlStream.Close()
			p.controlStream = nil
		}
		if p.videoStream != nil {
			p.videoStream.Close()
			p.videoStream = nil
		}
		if p.audioStream != nil {
			p.audioStream.Close()
			p.audioStream = nil
		}
		p.Speed.showSpeed = false
	}()
	p.quitChan = quit
	originalCmd = p.AdbClient.Cmd
	r := gin.Default()
	r.GET("/", p.HandleWebStream)
	r.GET("/video-proxy/:port", p.HandleWebStream)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", p.LocalPort),
		Handler: r,
	}
	go func() {
		log.Printf("[%s]video web server started\n", p.AdbClient.abstractSerial())
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

func (p *VideoProxy) StartProxy() {
	videoQuit := make(chan int, 1)
	audioQuit := make(chan int, 1)

	go p.AdbClient.StartVideo()
	go p.videoProxy(videoQuit)
	go p.audioProxy(audioQuit)

	p.coldSignalChan = make(chan int, 1)
	select {
	case <-p.coldSignalChan:
		videoQuit <- 1
		audioQuit <- 1
	}
}

func (p *VideoProxy) Connect() error {

	//dumpsys window policy|grep mIsShowing  if false is unlock

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", p.RemotePort))
	if err != nil {
		log.Printf("Can't connect to screen port %d \n", p.RemotePort)
		return err
	}
	log.Printf("[%s]connect phone success\n", p.AdbClient.abstractSerial())
	p.videoStream = conn

	p.deviceQuitChan = make(chan int, 1)
	go p.HandleVideoStream(p.deviceQuitChan)

	conn, err = net.Dial("tcp", fmt.Sprintf(":%d", p.ControlPort))
	if err != nil {
		log.Printf("[%s]Can't connect to control port %d \n", p.AdbClient.abstractSerial(), p.ControlPort)
		return err
	}
	p.controlStream = conn
	log.Printf("[%s]connect control success\n", p.AdbClient.abstractSerial())

	conn, err = net.Dial("tcp", fmt.Sprintf(":%d", p.AudioProxy.RemotePort))
	if err != nil {
		log.Printf("[%s]Can't connect to audio port %d \n", p.AdbClient.abstractSerial(), p.AudioProxy.RemotePort)
		return err
	}
	p.audioStream = conn
	log.Printf("[%s]connect audio success\n", p.AdbClient.abstractSerial())
	p.deviceAudioQuitChan = make(chan int, 1)
	go p.HandleAudioStream(p.deviceAudioQuitChan)

	return nil
}

func (p *VideoProxy) HandleVideoStream(quit chan int) {
	for p.videoStream != nil {
		select {
		case <-quit:
			log.Println("Quit device video stream")
			return
		default:
			if !p.DeviceFrameParse() {
				log.Printf("parse video frame error")
				log.Printf("video stream is nil %v", p.videoStream == nil)
			}
		}

	}
}

func (p *VideoProxy) DeviceFrameParse() bool {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s]:Video frame parse error %s\n", p.AdbClient.abstractSerial(), err)
			return
		}
	}()
	if p.videoStream == nil {
		return false
	}
	chunk := make([]byte, 10240)
	cnt, err := p.videoStream.Read(chunk)
	if err != nil {
		if err != io.EOF {
			log.Println("Got Error when receive from device: ", err)
			return false
		}
		return true
	}
	if p.showSpeed {
		p.serverBytes += float64(cnt)
	}
	chunk = chunk[:cnt]
	// banner only send once in the beginning
	if len(chunk) >= 24 && !p.videoData.hasBanner {
		p.videoData.hasBanner = true
		banner := &Banner{}
		banner.version = int(chunk[0])
		banner.length = int(chunk[1])
		banner.pid = binary.LittleEndian.Uint32(chunk[2:6])
		banner.realWidth = binary.LittleEndian.Uint32(chunk[6:10])
		banner.realHeight = binary.LittleEndian.Uint32(chunk[10:14])
		banner.virtualWidth = binary.LittleEndian.Uint32(chunk[14:18])
		banner.virtualHeight = binary.LittleEndian.Uint32(chunk[18:22])
		banner.orientation = int(chunk[22]) * 90
		banner.quirks = int(chunk[23])
		if banner.orientation > 0 {
			banner.realWidth, banner.realHeight = banner.realHeight, banner.realWidth
		}
		//log.Println(banner)
		p.videoData.banner = banner
		chunk = chunk[24:]
	}

	// parse rotation
	if len(chunk) >= 8 && chunk[0] == 0x04 && chunk[1] == 0x00 {
		//log.Println("rotation")
		data := chunk[:8]
		chunk = chunk[8:]
		p.videoData.hasRotation = true
		if p.screenStream != nil && p.videoData.videoMode {
			msg := map[string]uint32{}
			msg["height"] = p.videoData.banner.realHeight
			msg["width"] = p.videoData.banner.realWidth
			msg["rotation"] = uint32(data[4])
			p.screenStream.WriteJSON(msg)
		}
	}

	if len(chunk) == 0 {
		return true
	}
	//image mode
	if !p.videoData.videoMode {
		//the last chunk leftover
		if len(p.imageData.frameBody) > 0 && p.imageData.length == 0 {
			chunk = append(p.imageData.frameBody, chunk...)
		}

		//image header
		if len(chunk) > 5 && chunk[4] == 0xFF && chunk[5] == 0xD8 {
			p.imageData.length = int(binary.LittleEndian.Uint32(chunk[:4]))
			chunk = chunk[4:]
			//chunk larger than one picture
			if p.imageData.length < len(chunk) {
				p.imageData.frameBody = chunk[:p.imageData.length]
				p.imageData.leftOver = chunk[p.imageData.length:]
				p.imageData.length = 0
			} else {
				p.imageData.frameBody = chunk
				p.imageData.length -= len(chunk)
			}
		} else {
			if len(chunk) > p.imageData.length {
				p.imageData.frameBody = append(p.imageData.frameBody, chunk[:p.imageData.length]...)
				p.imageData.leftOver = chunk[p.imageData.length:]
				p.imageData.length = 0
			} else {
				p.imageData.frameBody = append(p.imageData.frameBody, chunk...)
				p.imageData.length -= len(chunk)
			}
		}
		if p.imageData.length == 0 && len(p.imageData.frameBody) > 1 {
			if p.imageData.frameBody[0] == 0xFF && p.imageData.frameBody[1] == 0xD8 {
				if p.screenStream != nil {
					msg := fmt.Sprintf("%d%s%s", time.Now().UnixNano()/1e6, "data:image/png;base64,", base64.StdEncoding.EncodeToString(p.imageData.frameBody))
					if p.screenStream != nil {
						p.mutex.Lock()
						p.screenStream.WriteMessage(1, []byte(msg))
						p.mutex.Unlock()
					}
				}
			} else {
				p.imageData.leftOver = nil
			}

			p.imageData.frameBody = p.imageData.leftOver
			p.imageData.length = 0
		}
	} else {
		if p.videoData.hasRotation {
			if p.screenStream != nil {
				p.mutex.Lock()
				p.screenStream.WriteMessage(2, chunk)
				p.mutex.Unlock()
			}
			p.videoData.frameBody = append(p.videoData.frameBody, chunk...)
			if len(p.videoData.frameBody) > 1024*1024*5 {
				p.videoData.frameBody = []byte{}
			}
		}
	}
	return true
}
