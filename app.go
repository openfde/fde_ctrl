package main

import (
	"encoding/base64"
	"io/fs"
	"io/ioutil"
	"net/http"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fde_ctrl/response"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
)

//scans applications in the linux.

const desktopEntryPath = "/usr/share/applications"
const iconPixmapPath = "/usr/share/pixmaps"
const iconsPath = "/usr/share/icons/hicolor/16x16"

type LinuxAppInterface interface {
	// Scan() error
	Setup(r *gin.RouterGroup)
}

type AppImpl struct {
	Type     string
	Path     string
	Icon     string
	IconPath string
	IconType string
	Name     string
	ZhName   string
}

type Apps []AppImpl

func (impls *Apps) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/apps", impls.ScanHandler)
}

func (impls *Apps) ScanHandler(c *gin.Context) {
	impls.Scan(iconPixmapPath, iconsPath, desktopEntryPath)
	c.JSON(http.StatusOK, response.Infra{
		Code:    200,
		Message: "success",
		Data:    impls,
	})
}

func (impls *Apps) Scan(iconPixmapPath, iconsPath, desktopEntryPath string) error {
	// 调用递归函数遍历目录下的所有文件
	err := filepath.Walk(desktopEntryPath, impls.visitEntries)
	if err != nil {
		return err
	}
	absPath := ""
	// var filterApps *Apps
	var filteredApps Apps
	for index, app := range *impls {
		absPath = ""
		//首先确定其是不是绝对路径,且有后缀
		if filepath.IsAbs(app.IconPath) && app.IconPath[0] == filepath.Separator && filepath.Ext(app.IconPath) != "" {
			_, err := os.Stat(app.IconPath)
			if os.IsNotExist(err) {
				//文件不存在，则跳过
			} else {
				absPath = app.IconPath
				(*impls)[index].readIconForApp(absPath, nil, nil)
			}
		} else {
			//寻找这个相对路径的文件
			//1 pixmap目录
			filepath.Walk(iconPixmapPath, (*impls)[index].readIconForApp)
			 if len((*impls)[index].Icon) == 0 {
			 	filepath.Walk(iconsPath,(*impls)[index].readIconForApp)
			 }
		}
		if len((*impls)[index].Icon) != 0 {
			fmt.Println("debug", (*impls)[index].Name)
			filteredApps = append(filteredApps, (*impls)[index])
		}
	}
	*impls = append((*impls)[:0])
	fmt.Println("debug", len(filteredApps))
	for _, app := range filteredApps {
		*impls = append(*impls, app)
	}

	return nil
}

func (impl *AppImpl) readIconForApp(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !strings.Contains(path, impl.IconPath) {
		return nil
	}
	//如果已经获取文件内容了，也退出
	if len(impl.Icon) > 0 {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	impl.Icon = base64.StdEncoding.EncodeToString(data)

	impl.IconType = filepath.Ext(path)
	impl.IconPath = path
	return nil
}

// 递归访问指定目录下的所有文件和子目录
func (impl *Apps) visitEntries(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !strings.Contains(path, ".desktop") {
		return nil
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return err
	}

	// 获取配置文件中的值
	section := cfg.Section("Desktop Entry")
	name := section.Key("Name").String()
	zhName := section.Key("Name[zh-CN]").String()
	iconPath := section.Key("Icon").String()
	execPath := section.Key("Exec").String()
	entryType := section.Key("Type").String()
	*impl = append(*impl, AppImpl{
		Type:     entryType,
		Path:     execPath,
		IconPath: iconPath,
		Name:     name,
		ZhName:   zhName,
	})
	return nil
}
