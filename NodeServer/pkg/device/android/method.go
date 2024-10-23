package android

import (
	"fmt"
	"github.com/electricbubble/gadb"
	"github.com/gridsystem-node/pkg/heartbeat"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"runtime"
	"time"
)

func CloseAllDevice() {
	for _, v := range AndroidDeviceMap {
		v.Close()
	}
}

func GetDeviceList() []heartbeat.DeviceInfo {
	var list []heartbeat.DeviceInfo
	for _, v := range AndroidDeviceMap {
		if v == nil {
			continue
		}
		model, _ := v.Device.Model()
		product, _ := v.Device.Product()
		state, _ := v.Device.State()
		list = append(list, heartbeat.DeviceInfo{
			Serial:   v.abstractSerial(),
			Model:    model,
			Product:  product,
			State:    string(state),
			Platform: "android",
		})
	}
	return list
}

func GetDevice(udid string) *AndroidDevice {
	return AndroidDeviceMap[udid]
}
func ColdInternal(udid string) {
	device := GetDevice(udid)
	ColdDeviceInternal(device)
}

func RemoveDevice(udid string) {
	delete(AndroidDeviceMap, udid)
}

var appStopExclude []string

func ColdDeviceInternal(device *AndroidDevice) {
	log.Printf("[%s]Releasing...\n", device.Device.Serial())

	device.SetSecret(IdGenerator(10))
	device.AppTerminateAll(appStopExclude...)
	resetDone := make(chan bool)
	go func() {
		if r := recover(); r != nil {
			log.Printf("Internal error: %v", r)
			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, false)
			log.Printf("%s\n", string(buf[0:stackSize]))
			CloseAllDevice()
			os.Exit(1)
		}
		device.Reset(true)
		resetDone <- true
	}()
	select {
	case <-resetDone:
	case <-time.After(time.Minute * 15):
		log.Printf("[%s]Reset failed after 15m.\n", device.abstractSerial())
	}

	device.SetSecret(IdGenerator(10))

	RemoveDevice(device.Device.Serial())
	heartbeat.GenerateHeartBeat(GetDeviceList())
	log.Printf("[%s]colding finished\n", device.abstractSerial())
}
func ResetAllDevices() {
	WaitAdbServerStarted()
	log.Printf("Resetting all devices...\n")
	for _, v := range AndroidDeviceMap {
		ColdDeviceInternal(v)
	}
	for udid := range AndroidDeviceMap {
		ColdInternal(udid)
	}
}
func WaitAdbServerStarted() {
	i := 0
	for {
		if i >= 5 {
			log.Printf("ADB server not started after %d seconds, exiting...", i)
			os.Exit(1)
		}
		con, err := net.Dial("tcp", fmt.Sprintf(":%d", gadb.AdbServerPort))
		if err != nil {
			log.Printf("ADB server not started, waiting...\n")
		} else {
			log.Printf("ADB server started\n")
			con.Close()
			return
		}
		i++
		time.Sleep(time.Second)
	}
}
