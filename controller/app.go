package controller

import (
	"encoding/base64"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

var defaultIconThemes = []string{"ukui-icon-theme-default", "bloom", "hicolor", "gnome", "bloom-dark", "Papirus"}
var defaultIconSizes = []string{"64x64", "scalable", "64", "apps/64", "places/64", "devices/64"}

// var iconPathList = []string{iconsHiColorPath, iconsGnomePath, iconsUKuiPath}

func (appImpl *Apps) Scan() {
	mutex.Lock()
	defer mutex.Unlock()
	if len(*appImpl) > 0 {
		*appImpl = make(Apps, 0)
	}
	appImpl.scan(iconOtherPathList, desktopEntryPath, false)
	return
}

type AppImpl struct {
	Type         string
	Path         string
	Icon         string
	IconPath     string
	IconType     string
	Name         string
	ZhName       string
	FileName     string
	IsAndroidApp bool
}

type Apps []AppImpl

type LinuxApps struct {
	Apps           //apps in the applications
	Shortcuts Apps //apps on the desktop
}

func (impl LinuxApps) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/apps", impl.Apps.ScanHandler)
	v1.GET("/desktopapps", impl.ScanDesktopHandler)
}

// func (impl Apps) Setup(r *gin.RouterGroup) {
// 	v1 := r.Group("/v1")
// 	v1.GET("/apps", impl.ScanHandler)
// }

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
		impls.Scan()
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

type personalPath struct {
	Desktop  string
	Document string
	Download string
	Music    string
	Picture  string
	Video    string
}

func getDefaultDesktopPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("get_home_dir_error", nil, err)
		return ""
	}
	defaultDesktopPath := filepath.Join(homeDir, "Desktop")
	_, err = os.Stat(defaultDesktopPath)
	if err == nil {
		return defaultDesktopPath
	}
	defaultDesktopPath = filepath.Join(homeDir, "桌面")
	_, err = os.Stat(defaultDesktopPath)
	if err == nil {
		return defaultDesktopPath
	}
	return ""
}

func getDesktopPath() (personalPath, error) {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("get_home_dir_error", nil, err)
		return personalPath{}, err
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		logger.Error("get_user_config_dir_error", nil, err)
		return personalPath{}, nil
	}
	desktopConfPath := filepath.Join(configDir, "user-dirs.dirs")
	content, err := ioutil.ReadFile(desktopConfPath)
	if err != nil {
		logger.Error("read_desktop_conf_error", nil, err)
		return personalPath{}, err
	}
	lines := strings.Split(string(content), "\n")
	var path personalPath
	for _, line := range lines {
		if strings.Contains(line, "XDG_DESKTOP_DIR") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				path.Desktop = strings.Trim(parts[1], "\"$HOME/")
				path.Desktop = filepath.Join(homeDir, path.Desktop)
			}
		}
	}
	return path, nil
}

func (linuxAppsImpl *LinuxApps) Scan() {
	linuxAppsImpl.Apps.Scan()
	linuxAppsImpl.ScanDesktop(false)
}

func (linuxAppsImpl *LinuxApps) ScanDesktop(withAndroid bool) {
	homeDir := os.Getenv("HOME")
	var desktopPath string
	personalPath, err := getDesktopPath()
	if err != nil {
		desktopPath = getDefaultDesktopPath()
		if desktopPath == "" {
			logger.Error("get_default_desktop_path_error", nil, err)
			desktopPath = filepath.Join(homeDir, "Desktop")
		}
	}
	desktopPath = personalPath.Desktop
	mutex.Lock()
	defer mutex.Unlock()
	if len(linuxAppsImpl.Shortcuts) > 0 { //reset the shortcuts
		linuxAppsImpl.Shortcuts = make([]AppImpl, 0)
	}
	linuxAppsImpl.Shortcuts.scan(iconOtherPathList, desktopPath, withAndroid)
	return
}

func (impls *LinuxApps) ScanDesktopHandler(c *gin.Context) {
	refresh := c.DefaultQuery("refresh", "false")
	withAndroid := c.DefaultQuery("withAndroid", "false")
	if refresh == "true" || refresh == "True" {
		logger.Info("scan_desktopapp_refresh", refresh)
		withAndroidBool := withAndroid == "true" || withAndroid == "True"
		impls.ScanDesktop(withAndroidBool)
	}
	pageQuery := getPageQuery(c)
	var data Apps
	pageQuery.Total = len(impls.Shortcuts)
	if pageQuery.PageEnable {
		start := (pageQuery.Page - 1) * pageQuery.PageSize
		end := start + pageQuery.PageSize
		start, end = validatePage(start, end, len(impls.Shortcuts))
		data = impls.Shortcuts[start:end]
	}
	response.ResponseWithPagination(c, pageQuery, data)
}

func (impls *Apps) scan(iconOtherPathList []string, desktopEntryPath string, withAndroid bool) {
	var iconPathList []string

	//add icon themes path into icon path list
	for index, _ := range defaultIconThemes {
		for sizeIndex, _ := range defaultIconSizes {
			iconPathList = append(iconPathList, iconPath+defaultIconThemes[index]+"/"+defaultIconSizes[sizeIndex])
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
		if !withAndroid {
			if strings.Contains(app.Path, "fde_launch") {
				continue
			}
		}
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
		if !strings.Contains(desktopEntryPath, baseDir) {
			*impls = append(*impls, AppImpl{
				FileName:     app.FileName,
				Type:         app.Type,
				Path:         app.Path,
				IconPath:     app.IconPath,
				IconType:     app.IconType,
				Name:         app.Name,
				ZhName:       app.ZhName,
				IsAndroidApp: app.IsAndroidApp,
			})
		} else {
			*impls = append(*impls, app)
		}
	}
	return
}

func (impl *AppImpl) readIconForApp(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	//unmatched file
	if !strings.Contains(path, impl.IconPath) {
		return nil
	}
	IconType := filepath.Ext(path)
	//only need the type of below
	if !(IconType == ".jpg" || IconType == ".jpeg" || IconType == ".svg" || IconType == ".png" || IconType == ".svgz") {
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
	if strings.Contains(strings.ToLower(name), "openfde") {
		return nil // skip OpenFDE
	}
	onlyShowIn := section.Key("OnlyShowIn").String()
	if onlyShowIn == "MATE" {
		return nil
	}
	notShowIn := section.Key("NotShowIn").String()
	if notShowIn == "OpenFDE" {
		return nil
	}

	zhName := section.Key("Name[zh_CN]").String()
	iconPath := section.Key("Icon").String()
	execPath := section.Key("Exec").String()
	entryType := section.Key("Type").String()
	noDisplay := section.Key("NoDisplay").String()
	if strings.Contains(noDisplay, "true") {
		return nil
	}
	*impl = append(*impl, AppImpl{
		FileName:     path,
		Type:         entryType,
		Path:         execPath,
		IconPath:     iconPath,
		Name:         name,
		ZhName:       zhName,
		IsAndroidApp: strings.Contains(execPath, "fde_launch"),
	})
	return nil
}
