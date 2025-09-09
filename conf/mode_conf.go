package conf

import (
	"fde_ctrl/logger"

	"github.com/go-ini/ini"
)

type ModeConf struct {
	Mode string //desktop or app_fusing
}

func ReadModeConf() (modeConf ModeConf, err error) {
	cfg, err := ini.Load("/etc/fde.d/fde.conf")
	if err != nil {
		logger.Error("fded_conf_error", nil, err)
		return modeConf, err
	}

	// 无section的键值对会被自动归到默认的""（空字符串）section中。
	defaultSection := cfg.Section("")

	modeConf.Mode = defaultSection.Key("mode").String()
	return modeConf, nil
}
