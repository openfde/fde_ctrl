package conf

import (
	"errors"
	"fde_ctrl/logger"
	"io/ioutil"

	"github.com/go-ini/ini"
)

const (
	sectionApp    = "App"
	sectonXserver = "Xserver"
	fusionApp     = "FusingApp"
)

type Xserver struct {
	WithOutTheme []string
}

type CServer struct {
	ClientName string //gnome-terminal
	ServerName string //gnome-terminal-server
}

type FusionApp struct {
	CServer []CServer
}

type App struct {
	IconSizes  []string //16 x 16 default
	IconThemes []string //hicolor
}

type Configure struct {
	App       App
	Xserver   Xserver
	FusionApp FusionApp
}

func Read() (configure Configure, err error) {
	cfg, err := ini.Load("/etc/fde.conf")
	if err != nil {
		logger.Error("load config", nil, err)
		return
	}

	// 获取配置文件中的值
	sectionApp := cfg.Section(sectionApp)
	configure.App.IconSizes = sectionApp.Key("IconSizes").Strings(",")
	configure.App.IconThemes = sectionApp.Key("IconThemes").Strings(",")

	sectionXserver := cfg.Section(sectonXserver)
	configure.Xserver.WithOutTheme = sectionXserver.Key("WithoutTheme").Strings(",")

	sectionFusionApp := cfg.Section(fusionApp)
	cserverList := sectionFusionApp.Key("CServer").Strings(",")
	if len(cserverList)%2 != 0 {
		logger.Error("cserver_list", len(cserverList), errors.New("cserver list is not even"))
		return
	}
	for i, v := range cserverList {
		if i%2 != 0 {
			continue
		}
		configure.FusionApp.CServer = append(configure.FusionApp.CServer, CServer{ClientName: v, ServerName: cserverList[i+1]})
	}
	//go to find configure which locate at  /etc/fde.d/
	//获取/etc/fde.d/下的所有文件
	files, err := ioutil.ReadDir("/etc/fde.d/")
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			//读取文件内容
			cfg, err = ini.Load("/etc/fde.d/" + file.Name())
			if err != nil {
				logger.Error("load config", file.Name(), err)
				err = nil
				return
			}
			sectionFusionApp := cfg.Section(fusionApp)
			cserverList := sectionFusionApp.Key("CServer").Strings(",")
			if len(cserverList)%2 != 0 {
				logger.Error("cserver_list", len(cserverList), errors.New("cserver list is not even "+file.Name()))
				return
			}
			for i, v := range cserverList {
				if i%2 != 0 {
					continue
				}
				configure.FusionApp.CServer = append(configure.FusionApp.CServer, CServer{ClientName: v, ServerName: cserverList[i+1]})
			}
			sectionXserver := cfg.Section(sectonXserver)
			configure.Xserver.WithOutTheme = append(configure.Xserver.WithOutTheme, sectionXserver.Key("WithoutTheme").Strings(",")...)
		}
	}
	err = nil
	return
}
