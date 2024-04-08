package main

import (
	"context"
	"errors"
	"fde_ctrl/logger"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type SysMethod struct {
}

const (
	maxBrightness     = "max_brightness"
	currentBrightness = "brightness"
)

func (impl *SysMethod) detect(mainCtx context.Context) (string, error) {
	// decide type by checking the existence of the backlight
	backDirPath := "/sys/class/backlight"

	files, err := ioutil.ReadDir(backDirPath)
	if err != nil {
		logger.Error("brightness_read_backlight", nil, err)
		return "", err
	}
	if len(files) == 0 {
		//means there is no backlinght backend
		return "", nil
	}
	var displayPath string
	for _, file := range files {
		fileName := file.Name()
		displayPath = filepath.Join(backDirPath, fileName)
		break //break after reading the first one
	}
	return displayPath, nil
}

func (impl *SysMethod) Detect(mainCtx context.Context) error {
	dir, err := impl.detect(mainCtx)
	if err != nil {
		logger.Error("set_brightness_sysfs", nil, err)
		return err
	}
	if len(dir) == 0 {
		err := errors.New("display in /sys/class/backlight not found")
		logger.Error("set_brightness_sysfs", nil, err)
		return err
	}
	fmt.Println("sys")
	return nil
}

func (impl *SysMethod) Set(mainCtx context.Context, bus, brightness string) error {
	dir, err := impl.detect(mainCtx)
	if err != nil {
		logger.Error("set_brightness_sysfs", nil, err)
		return err
	}
	if len(dir) == 0 {
		err := errors.New("display in /sys/class/backlight not found")
		logger.Error("set_brightness_sysfs", nil, err)
		return err
	}

	return ioutil.WriteFile(filepath.Join(dir, currentBrightness), []byte(brightness), 0644)
}

func (impl *SysMethod) Get(mainCtx context.Context, bus string) (err error) {
	dir, err := impl.detect(mainCtx)
	if err != nil {
		logger.Error("get_brightness_sysfs", nil, err)
		return err
	}
	if len(dir) == 0 {
		err := errors.New("display in /sys/class/backlight not found")
		logger.Error("get_brightness_sysfs", nil, err)
		return err
	}
	var max, current string
	content, err := ioutil.ReadFile(filepath.Join(dir, maxBrightness))
	if err != nil {
		logger.Error("read_max_brightness_sys", maxBrightness, err)
		return
	}

	max = strings.ReplaceAll(string(content), "\n", "")
	// 将文件内容转换为字符串并打印
	content, err = ioutil.ReadFile(filepath.Join(dir, currentBrightness))
	if err != nil {
		logger.Error("get_brightness_sysfs", nil, err)
		return
	}
	current = strings.ReplaceAll(string(content), "\n", "")
	fmt.Println(current, max)
	return
}
