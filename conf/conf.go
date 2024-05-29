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
	sectionApp        = "App"
)

const (
	sectionFusion = "PersonalDirFusing"
)

type App struct {
	IconSizes  []string //16 x 16 default
	IconThemes []string //hicolor
}

type Android struct {
	Image string
}

type Http struct {
	Host string
}

// func (win WindowsManager) IsWayland() bool {
// 	//actually fde_wm is renamed from mutter, because mutter is a protected process name on kylin operator system
// 	return win.Protocol == "wayland" || win.Name == "fde_wm"
// }

type Display struct {
	Resolution string
}

type Configure struct {
	Android Android
	Display Display
	Http    Http
	App     App
}

type PersonalDirFusing struct {
	Fusing bool
}

type CustomerConfigure struct {
	PersonalDirFusing PersonalDirFusing
}

func Read() (configure Configure, customerConfigure CustomerConfigure, err error) {
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

	sectionDisplay := cfg.Section(sectionDisplay)
	configure.Display.Resolution = sectionDisplay.Key("Resolution").String()
	if len(configure.Display.Resolution) == 0 {
		configure.Display.Resolution = "1920,1080"
	}
	sectionApp := cfg.Section(sectionApp)
	configure.App.IconSizes = sectionApp.Key("IconSizes").Strings(",")
	configure.App.IconThemes = sectionApp.Key("IconThemes").Strings(",")

	cfg, err = ini.Load("/etc/fde.d/customer.conf")
	if err != nil {
		logger.Error("load config", nil, err)
		return
	}
	sectionFusion := cfg.Section(sectionFusion)
	customerConfigure.PersonalDirFusing.Fusing = sectionFusion.Key("Fusing").MustBool()
	return

}
