package adb

import (
	"fmt"
	"github.com/gridsystem-node/pkg/device/android"
	"log"
	"os"
	"os/exec"
)

var Done bool

func StartAdbServer() {
	for {
		cmd := exec.Command(android.AdbPath(), "-a", "nodaemon", "server", "start")
		err := cmd.Run()
		if err != nil {
			if Done {
				return
			}
			log.Println("[ADB]: error start adb server", err)
			os.Exit(1)
		}
		if Done {
			return
		}
		android.CloseAllDevice()
		log.Println("[ADB]: adb server restart")
		go android.ResetAllDevices()
		err = KillAdbServer()
		if err != nil {
			log.Println("kill adb server error", err)
			log.Println("please run adb kill-server before restart")
			os.Exit(1)
		}
	}
}
func KillAdbServer() error {
	output, err := exec.Command(android.AdbPath(), "kill-server").CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return err
	}
	return nil
}
