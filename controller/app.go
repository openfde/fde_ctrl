package controller

import (
	"encoding/base64"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/response"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
)

//scans applications in the linux.

const baseDir = "/usr/share"
const desktopEntryPath = baseDir + "/applications"
const iconPixmapPath = baseDir + "/pixmaps"
const iconKylinCenterPath = baseDir + "/kylin-software-center/data/icons/"
const iconPath = baseDir + "/icons/"

var iconOtherPathList = []string{iconPixmapPath, iconKylinCenterPath}

var defaultIconThemes = []string{"hicolor", "ukui-icon-theme-deafult", "gnome"}
var defaultIconSizes = []string{"64x64", "scalable"}

// var iconPathList = []string{iconsHiColorPath, iconsGnomePath, iconsUKuiPath}

var config conf.Configure

func (appImpl *Apps) Scan(configure conf.Configure) {
	mutex.Lock()
	defer mutex.Unlock()
	if len(*appImpl) > 0 {
		*appImpl = make(Apps, 0)
	}
	config = configure
	appImpl.scan(iconOtherPathList, desktopEntryPath, configure.App.IconThemes, configure.App.IconSizes)
	return
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

func (impl Apps) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/apps", impl.ScanHandler)
}

func validatePage(start, end, length int) (int, int) {
	switch {
	case start > length:
		{
			start = length
			end = start
		}
	case end > length || start > end:
		{
			end = length
		}
	}
	return start, end
}

var mutex = &sync.Mutex{}

func (impls *Apps) ScanHandler(c *gin.Context) {
	refresh := c.DefaultQuery("refresh", "false")
	if refresh == "true" || refresh == "True" {
		logger.Info("scan_app_refresh", refresh)
		impls.Scan(config)
	}
	pageQuery := getPageQuery(c)
	var data Apps
	pageQuery.Total = len(*impls)
	if pageQuery.PageEnable {
		start := (pageQuery.Page - 1) * pageQuery.PageSize
		end := start + pageQuery.PageSize
		start, end = validatePage(start, end, len(*impls))
		data = (*impls)[start:end]
	}
	response.ResponseWithPagination(c, pageQuery, data)
}

func (impls *Apps) scan(iconOtherPathList []string, desktopEntryPath string, iconThemes, iconSizes []string) {
	var iconPathList []string

	if len(iconSizes) == 0 || (len(iconSizes) == 1 && iconSizes[0] == "") {
		iconSizes = defaultIconSizes
	}
	if len(iconThemes) == 0 || (len(iconThemes) == 1 && iconThemes[0] == "") {
		iconThemes = defaultIconThemes
	}
	//add icon themes path into icon path list
	for index, _ := range iconThemes {
		for sizeIndex, _ := range iconSizes {
			iconPathList = append(iconPathList, iconPath+iconThemes[index]+"/"+iconSizes[sizeIndex])
		}
	}
	//add pixmap path into icon path list
	iconPathList = append(iconPathList, iconOtherPathList...)

	// 调用递归函数遍历目录下的所有文件
	filepath.Walk(desktopEntryPath, impls.visitDesktopEntries)
	absPath := ""
	// var filterApps *Apps
	var filteredApps Apps
	for index, app := range *impls {
		absPath = ""
		//首先确定其是不是绝对路径,且有后缀
		if filepath.IsAbs(app.IconPath) && filepath.Ext(app.IconPath) != "" {
			_, err := os.Stat(app.IconPath)
			if os.IsNotExist(err) {
				//文件不存在，则跳过
			} else {
				absPath = app.IconPath
				(*impls)[index].readIconForApp(absPath, nil, nil)
			}
		} else {
			//寻找这个相对路径的文件
			for _, pathValue := range iconPathList {
				_, err := os.Stat(pathValue)
				if os.IsNotExist(err) {
					continue
				}
				filepath.Walk(pathValue, (*impls)[index].readIconForApp)
				if len((*impls)[index].Icon) > 0 {
					break
				}
			}
			// filepath.Walk(iconPixmapPath, (*impls)[index].readIconForApp)
			// if len((*impls)[index].Icon) == 0 {
			// 	filepath.Walk(iconsPath, (*impls)[index].readIconForApp)
			// }
		}
		if len((*impls)[index].Icon) != 0 {
			filteredApps = append(filteredApps, (*impls)[index])
		}
	}
	*impls = append((*impls)[:0])
	for _, app := range filteredApps {
		*impls = append(*impls, app)
	}

	return
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
	if len(impl.IconType) == 0 && strings.Contains(path, "pixmaps") {
		impl.IconType = ".png"
	}
	impl.IconPath = path
	return nil
}

// 递归访问指定目录下的所有文件和子目录
func (impl *Apps) visitDesktopEntries(path string, info fs.FileInfo, err error) error {
	if err != nil {
		logger.Error("visit_lstat_desktop_file", path, err)
		return nil
	}
	if !strings.Contains(path, ".desktop") {
		return nil
	}

	cfg, err := ini.Load(path)
	if err != nil {
		logger.Error("load_desktop_file", path, err)
		return nil // skip invalid desktop file
	}

	// 获取配置文件中的值
	section := cfg.Section("Desktop Entry")
	name := section.Key("Name").String()
	if name == "OpenFDE" {
		return nil // skip OpenFDE
	}
	zhName := section.Key("Name[zh-CN]").String()
	iconPath := section.Key("Icon").String()
	execPath := section.Key("Exec").String()
	entryType := section.Key("Type").String()
	noDisplay := section.Key("NoDisplay").String()
	if strings.Contains(noDisplay, "true") {
		return nil
	}
	*impl = append(*impl, AppImpl{
		Type:     entryType,
		Path:     execPath,
		IconPath: iconPath,
		Name:     name,
		ZhName:   zhName,
	})
	return nil
}
