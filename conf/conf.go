package conf

import (
	"fde_ctrl/logger"

	"github.com/go-ini/ini"
)

const (
	sectionAndroid    = "Android"
	sectionHttp       = "Http Server"
	sectionWinManager = "WindowsManager"
	sectionDisplay    = "Display"
)

type Android struct {
	Image string
}

type Http struct {
	Host string
}

type WindowsManager struct {
	Name string
}
type Display struct {
	Resolution string
}

type Configure struct {
	Android        Android
	WindowsManager WindowsManager
	Display        Display
	Http           Http
}

func Read() (configure Configure, err error) {
	cfg, err := ini.Load("/etc/fde.conf")
	if err != nil {
		logger.Error("load config", nil, err)
		return
	}

	// 获取配置文件中的值
	sectionAndroid := cfg.Section(sectionAndroid)
	configure.Android.Image = sectionAndroid.Key("Image").String()
	if len(configure.Android.Image) == 0 {
		configure.Android.Image = "fde:latest"
	}

	sectionHttp := cfg.Section(sectionHttp)
	configure.Http.Host = sectionHttp.Key("Host").String()
	if len(configure.Http.Host) == 0 {
		configure.Http.Host = "128.128.0.1"
	}

	sectionWinManager := cfg.Section(sectionWinManager)
	configure.WindowsManager.Name = sectionWinManager.Key("Name").String()
	if len(configure.WindowsManager.Name) == 0 {
		configure.WindowsManager.Name = "kwin"
	}

	sectionDisplay := cfg.Section(sectionDisplay)
	configure.Display.Resolution = sectionDisplay.Key("Resolution").String()
	if len(configure.Display.Resolution) == 0 {
		configure.Display.Resolution = "1920,1080"
	}
	return
}
