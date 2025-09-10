package conf

import (
	"fde_ctrl/logger"

	"github.com/go-ini/ini"
)

type NaviMode string

const (
	NaviModeHidden NaviMode = "hidden"
	NaviModeNormal NaviMode = "normal"
)

type ModeConf struct {
	NaviMode NaviMode //hidden or normal ； defalut is normal
}

func ReadModeConf() (modeConf ModeConf, err error) {
	cfg, err := ini.Load("/etc/fde.d/fde.conf")
	if err != nil {
		logger.Error("fded_conf_error", nil, err)
		return modeConf, err
	}

	// 无section的键值对会被自动归到默认的""（空字符串）section中。
	defaultSection := cfg.Section("")

	modeConf.NaviMode = NaviMode(defaultSection.Key("navi_mode").String())
	return modeConf, nil
}

func NaviModeIsHidden(mode NaviMode) bool {
	return mode == NaviModeHidden
}
