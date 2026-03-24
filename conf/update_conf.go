package conf

import (
	"os"
	"path/filepath"

	"github.com/go-ini/ini"
)

func WriteUpdatePolicy(currentVersion, debFile, updatePolicy string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	policyPath := filepath.Join(configDir, "fde_update.policy")

	cfg := ini.Empty()
	sec := cfg.Section("Update")
	sec.Key("CurrentVersion").SetValue(currentVersion)
	sec.Key("DebFile").SetValue(debFile)
	sec.Key("UpdatePolicy").SetValue(updatePolicy)

	return cfg.SaveTo(policyPath)
}

func ReadUpdatePolicy() (currentVersion, debFile, updatePolicy string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}

	policyPath := filepath.Join(home, ".config", "fde_update.policy")
	cfg, err := ini.Load(policyPath)
	if err != nil {
		return "", "", "", err
	}

	sec := cfg.Section("Update")
	currentVersion = sec.Key("CurrentVersion").String()
	debFile = sec.Key("DebFile").String()
	updatePolicy = sec.Key("UpdatePolicy").String()

	return currentVersion, debFile, updatePolicy, nil
}

func ReadCurrentVersion() (currentVersion string, err error) {
	const imagesPy = "/usr/lib/waydroid/tools/helpers/images.py"
	_, err = os.Stat(imagesPy)
	if err != nil && os.IsNotExist(err) {
		return "uninstalled", nil
	}
	f, err := os.Open(imagesPy)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 匹配示例: ro.openfde.version=1.2.3-20260323
	re := regexp.MustCompile(`ro\.openfde\.version=([0-9A-Za-z._-]+)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		m := re.FindStringSubmatch(line)
		if len(m) == 2 {
			return m[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("ro.openfde.version not found in images.py")
}
