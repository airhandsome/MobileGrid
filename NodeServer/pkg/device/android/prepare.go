package android

import (
	"github.com/electricbubble/gadb"
	log "github.com/sirupsen/logrus"
	"time"
)

var androidCli gadb.Client
var AndroidDeviceMap map[string]*AndroidDevice

func init() {
	var err error
	androidCli, err = gadb.NewClient()
	if err != nil {
		log.Debugf("init gadb client failed: %v", err)
	}
}

func WatchDevice(stop <-chan string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Debugf("stop watch device")
			return
		case <-ticker.C:
			device, err := androidCli.DeviceList()
			if err != nil {
				log.Debugf("get device list failed: %v", err)
				return
			}
			newDeviceMap := make(map[string]*AndroidDevice)
			for _, d := range device {
				newDeviceMap[d.Serial()] = &AndroidDevice{
					Device: &d,
				}
				if dev, ok := AndroidDeviceMap[d.Serial()]; !ok {
					go dev.prepare()
				}
			}
			AndroidDeviceMap = newDeviceMap
		}
	}
}
