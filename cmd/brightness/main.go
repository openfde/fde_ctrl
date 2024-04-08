/*
* Copyright （C)  2023 OpenFDE , All rights reserved.
 */

package main

import (
	"context"
	"fde_ctrl/logger"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var _version_ = "v0.1"
var _tag_ = "v0.1"
var _date_ = "20230101"

type Brightness interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) error
	Detect(context.Context) error
}

const (
	BrightnessErrorBusInvalid = 2
)

func main() {

	var version, help bool
	var mode, bus, brightness string
	flag.BoolVar(&version, "version", false, "print version")
	flag.BoolVar(&help, "help", false, "print help")
	flag.StringVar(&mode, "mode", "detect", "decide to detect|set|get")
	flag.StringVar(&bus, "bus", "11", "i2c bus")
	flag.StringVar(&brightness, "brightness", "80", "adjust breightness")
	flag.Parse()
	if help {
		fmt.Println("fde_brightness:")
		fmt.Println("\t-v: print versions and tags")
		fmt.Println("\t-h: print help")
		fmt.Println("\t-m: input the running mode[detect|get|set]")
		return
	}

	if version {
		fmt.Printf("Version: %s, tag: %s , date: %s \n", _version_, _tag_, _date_)
		return
	}
	var err error
	displayPath, err := decideType()
	if err != nil {
		logger.Error("decide_brightness_type", nil, err)
	}
	var brightnessImpl Brightness
	if strings.Compare(bus, "sys") == 0 && len(displayPath) == 0 {
		logger.Error("compare_bus_sys", displayPath, nil)
		os.Exit(BrightnessErrorBusInvalid) // 2 means wrong bus type
	} else if strings.Compare(bus, "sys") != 0 && len(displayPath) > 0 {
		logger.Error("compare_sys_bus", bus, nil)
		os.Exit(BrightnessErrorBusInvalid)
	}
	if len(displayPath) > 0 {
		brightnessImpl = new(SysMethod)
	} else {
		brightnessImpl = new(DDcutil)
	}
	mainCtx, _ := context.WithCancel(context.Background())
	switch {
	case mode == "detect":
		{
			err = brightnessImpl.Detect(mainCtx)
		}
	case mode == "get":
		{
			err = brightnessImpl.Get(mainCtx, bus)
		}
	case mode == "set":
		{
			err = brightnessImpl.Set(mainCtx, bus, brightness)
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func decideType() (string, error) {
	//decide type by checking the existence of the backlight
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
