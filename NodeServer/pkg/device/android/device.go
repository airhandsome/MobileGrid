package android

import (
	"fmt"
	"github.com/electricbubble/gadb"
	"github.com/gridsystem-node/pkg/util"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type AndroidDevice struct {
	Device *gadb.Device

	// handle video stream
	videoPort  int
	audioPort  int
	devicePath string
	videoProxy *VideoProxy
	AudioProxy *AudioProxy
	AdbProxy   *AdbConnection

	// params
	secret string
}

func (d *AndroidDevice) SetSecret(secret string) {
	d.secret = secret
}

func (d *AndroidDevice) GetSecret() string {
	return d.secret
}

func (d *AndroidDevice) abstractSerial() string {
	return d.Device.Serial()
}

func (d *AndroidDevice) prepare() {
	defer func() {
		//catch panic, case of device disconnect
		if r := recover(); r != nil {
			log.Debugf("prepare device %s panic: %v", d.Device.Serial(), r)
		}
	}()
	d.InitAction()
	d.InitFile()
	d.InitNetwork()
}

func (d *AndroidDevice) InitAction() {
	d.Device.RunShellCommand("input keyevent HOME")
	d.Device.RunShellCommand("input keyevent 164")
	d.Device.RunShellCommand("setting put system screen_brightness_mode 0")
	d.Device.RunShellCommand("setting put system screen_brightness 0")
	d.Device.RunShellCommand("settings put system screen_brightness_mode 0")
	d.Device.RunShellCommand("settings put system screen_brightness 0")
	d.Device.RunShellCommand("content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:1")
	d.Device.RunShellCommand("rm /data/local/tmp/*.apk")
	d.Device.RunShellCommand("logcat --clear")
}

func (d *AndroidDevice) InitFile() {
	prefix := filepath.Join(util.GetCurrentDir(), "plugin/")
	svideo, err := os.Open(prefix + "svideo")
	err = d.Device.PushFile(svideo, "/data/local/tmp/svideo.jar")
	if err != nil {
		return
	}
}

func (d *AndroidDevice) InitNetwork() {
	d.videoPort, d.audioPort = d.StartRunningProxy("svideo", "svideo-control", "saudio")
}
func (d *AndroidDevice) StartRunningProxy(videoPortName, controlPortName, audioPortName string) (int, int) {
	d.ensureVideoPortClean("svideo")
	videoPort := d.adbForwardToAny("localabstract:" + videoPortName)
	controlPort := d.adbForwardToAny("localabstract:" + controlPortName)
	audioPort := d.adbForwardToAny("localabstract:" + audioPortName)

	videoListenPort, err := util.GetFreePort()
	if err != nil {
		panic(err)
	}
	audioListenPort, err := util.GetFreePort()
	if err != nil {
		panic(err)
	}
	path, err := d.Device.RunShellCommand("CLASSPATH=/data/local/tmp/svideo.jar app_process / com.genymobile.scrcpy.Server -L")
	if err != nil {
		panic(err)
	}
	d.devicePath = strings.TrimSpace(path)

	args := fmt.Sprintf(
		" ANDROID_DATA=/data/local/tmp LD_LIBRARY_PATH=%s:/data/local/tmp CLASSPATH=/data/local/tmp/svideo.jar app_process / com.genymobile.scrcpy.Server -m %s",
		d.devicePath, "video")

	adbclient := AdbConnection{Device: d.Device, AbstractSerial: d.Device.Serial(), Cmd: args}

	tcpProxy := TcpProxy{LocalPort: videoListenPort, RemoteHost: "localhost", RemotePort: videoPort}
	audioProxy := &AudioProxy{TcpProxy: TcpProxy{LocalPort: audioListenPort, RemoteHost: "localhost", RemotePort: audioPort}}
	videoProxy := &VideoProxy{ControlPort: controlPort, UploadVideo: true, AdbClient: adbclient, TcpProxy: tcpProxy, AudioProxy: *audioProxy}
	go videoProxy.StartProxy()

	d.AudioProxy = audioProxy
	d.videoProxy = videoProxy
	return videoListenPort, audioListenPort
}

func (d *AndroidDevice) adbForwardToAny(remote string) int {
	// if already forwarded, just return
	forwards, err := d.Device.ForwardList()
	if err == nil {
		for _, f := range forwards {
			if f.Serial == d.Device.Serial() {
				if f.Remote == remote && strings.HasPrefix(f.Local, "tcp:") {
					result, err := strconv.Atoi(strings.TrimLeft(f.Local, "tcp:"))
					if err != nil {
						panic(err)
					}
					return result
				}
			}
		}
	} else {
		log.Printf("Error getting forward list: %s\n", err.Error())
	}

	localPort, err := util.GetFreePort()
	if err != nil {
		return -1
	}
	err = d.Forward(fmt.Sprintf("tcp:%d", localPort), remote, false)
	if err != nil {
		panic(err)
	}

	return localPort
}

func (d *AndroidDevice) ensureVideoPortClean(abstractPort string) {
	res, _ := d.Device.RunShellCommand("lsof | grep " + abstractPort)

	res = strings.TrimSpace(res)
	if res == "" {
		return
	}
	r := regexp.MustCompile("[^\\s]+")
	line := r.FindAllString(res, -1)
	if len(line) > 1 {
		pid := line[1]
		res, err := d.Device.RunShellCommand("kill -9 " + pid)
		log.Printf("[%s]kill %s returns: %s\n", d.Device.Serial(), pid, res)
		if err != nil {
			log.Println(err)
		}
	}
}

func (d *AndroidDevice) Forward(local string, remote string, noRebind ...bool) (err error) {
	command := ""

	if len(noRebind) != 0 && noRebind[0] {
		command = fmt.Sprintf("host-serial:%s:forward:norebind:%s;%s", d.Device.Serial(), local, remote)
	} else {
		command = fmt.Sprintf("host-serial:%s:forward:%s;%s", d.Device.Serial(), local, remote)
	}
	_, err = d.Device.RunShellCommand(command)
	return
}

func (d *AndroidDevice) Close() {
	log.Printf("[%s]Closing...\n", d.Device.Serial())

	d.AdbProxy.StopProxy()
	d.AdbProxy = nil

	d.videoProxy.StopProxy()
	d.videoProxy = nil
}

func (d *AndroidDevice) Terminate(packageName string) (err error) {
	_, err = d.Device.RunShellCommand("am force-stop", packageName)
	return
}

func (d *AndroidDevice) AppLaunch(packageName string) (err error) {
	var output string
	if output, err = d.Device.RunShellCommand("monkey -p", packageName, "-c android.intent.category.LAUNCHER 1"); err != nil {
		return err
	}
	if strings.Contains(output, "monkey aborted") {
		return fmt.Errorf("app launch: %s", strings.TrimSpace(output))
	}

	return
}
func (d *AndroidDevice) AppListRunning() []string {
	output, err := d.Device.RunShellCommand("pm", "list", "packages")
	if err != nil {
		return nil
	}
	reg := regexp.MustCompile(`package:(\S+)`)
	packageNames := reg.FindAllStringSubmatch(output, -1)

	reg = regexp.MustCompile(`(\S+)\r*\n`)
	output, err = d.Device.RunShellCommand("ps; ps -A")
	if err != nil {
		return nil
	}
	processNames := reg.FindAllStringSubmatch(output, -1)

	var result []string
	for _, pkg := range packageNames {
		for _, prs := range processNames {
			if pkg[1] == prs[1] {
				result = append(result, pkg[1])
				break
			}
		}
	}
	return result
}

func (d *AndroidDevice) AppTerminateAll(excludePackages ...string) {
	runningApps := d.AppListRunning()
	for _, r := range runningApps {
		if !contains(excludePackages, r) {
			d.Terminate(r)
		}
	}
}
func (d *AndroidDevice) AppUninstall(appPackageName string, keepDataAndCache ...bool) (err error) {
	var shellOutput string
	if len(keepDataAndCache) != 0 && keepDataAndCache[0] {
		shellOutput, err = d.Device.RunShellCommand("pm uninstall", "-k", appPackageName)
	} else {
		shellOutput, err = d.Device.RunShellCommand("pm uninstall", appPackageName)
	}

	if err != nil {
		return fmt.Errorf("apk uninstall: %w", err)
	}

	if !strings.Contains(shellOutput, "Success") {
		return fmt.Errorf("apk uninstalled: %s", shellOutput)
	}

	return
}
func (d *AndroidDevice) AppInstall(apkPath string, flags []string, reinstall ...bool) (err error) {
	apkName := filepath.Base(apkPath)
	if !strings.HasSuffix(strings.ToLower(apkName), ".apk") {
		return fmt.Errorf("apk file must have an extension of '.apk': %s", apkPath)
	}

	var apkFile *os.File
	if apkFile, err = os.Open(apkPath); err != nil {
		return fmt.Errorf("apk file: %w", err)
	}

	remotePath := path.Join(DeviceTempPath, apkName)
	if err = d.Device.PushFile(apkFile, remotePath); err != nil {
		return fmt.Errorf("apk push: %w", err)
	}

	var shellOutput string
	if flags != nil && len(flags) > 0 {
		shellOutput, err = d.Device.RunShellCommand("pm install", append(flags, remotePath)...)
	} else if len(reinstall) != 0 && reinstall[0] {
		shellOutput, err = d.Device.RunShellCommand("pm install", "-r", remotePath)
	} else {
		shellOutput, err = d.Device.RunShellCommand("pm install", remotePath)
	}

	if err != nil {
		return fmt.Errorf("apk install: %w", err)
	}

	if !strings.Contains(shellOutput, "Success") {
		return fmt.Errorf("apk installed: %s", shellOutput)
	}

	return
}

func (d *AndroidDevice) RunShellCmd(cmd string, args ...string) (string, error) {
	command, err := d.Device.RunShellCommand(cmd, args...)
	return command, err
}
func (d *AndroidDevice) Reset(close bool) {
	if close {
		d.Close()
	}

	//if !appconfig.IsPrivate {
	//	packages := d.getPackages()
	//	if packages != nil {
	//		whitelist, err := GetCosPackageWhitelist(d.abstractSerial())
	//		if err != nil || len(whitelist) == 0 {
	//			whitelist = appconfig.GetPackageWhitelist(d.abstractSerial())
	//		}
	//		if whitelist != nil {
	//			whitelist = append(whitelist, appconfig.WhitelistPackages...)
	//			if d.IsWhiteList {
	//				whitelist = append(whitelist, appconfig.WhitelistPackagesForWhiteListDevices...)
	//			}
	//			for _, pkg := range packages {
	//				p := strings.ReplaceAll(pkg, "\r", "")
	//				if !utils.SliceContains(p, whitelist) {
	//					log.Printf("[%s]Ready to uninstall %s\n", d.abstractSerial(), p)
	//					d.RunShellCmd("pm uninstall " + strings.Split(p, ":")[1])
	//				}
	//			}
	//		}
	//	}
	//}

	d.RunShellCmd("input keyevent HOME")
	d.RunShellCmd("input keyevent 164")
	d.RunShellCmd("setting put system screen_brightness_mode 0")
	d.RunShellCmd("setting put system screen_brightness 0")
	d.RunShellCmd("settings put system screen_brightness_mode 0")
	d.RunShellCmd("settings put system screen_brightness 0")
	d.RunShellCmd("content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:1")
	d.RunShellCmd("rm /data/local/tmp/*.apk")
	d.RunShellCmd("logcat --clear")

}
