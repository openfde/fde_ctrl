package conf

import (
	"github.com/go-ini/ini"
)

type FusingMode string

const (
	FusingModeDesktop   FusingMode = "desktop"
	FusingModeAppFusing FusingMode = "app_fusing"
)

type ModeConf struct {
	Mode FusingMode //； defalut is desktop
}

func ReadModeConf() (modeConf ModeConf, err error) {
	cfg, err := ini.Load("/etc/fde.d/fde.conf")
	if err != nil {
		//logger.Error("fded_conf_error", nil, err)
		return modeConf, err
	}

	// 无section的键值对会被自动归到默认的""（空字符串）section中。
	defaultSection := cfg.Section("")

	modeConf.Mode = FusingMode(defaultSection.Key("mode").String())
	return modeConf, nil
}

func IsFusingMode(mode FusingMode) bool {
	return mode == FusingModeAppFusing
}
