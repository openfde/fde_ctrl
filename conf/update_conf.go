package conf

import (
	"fde_ctrl/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
)

const SectionUpdate = "Update"
const KeyCurrentVersion = "CurrentVersion"
const KeyDebFile = "DebFile"
const KeyUpdatePolicy = "UpdatePolicy"

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
	sec := cfg.Section(SectionUpdate)
	sec.Key(KeyCurrentVersion).SetValue(currentVersion)
	sec.Key(KeyDebFile).SetValue(debFile)
	sec.Key(KeyUpdatePolicy).SetValue(updatePolicy)

	return cfg.SaveTo(policyPath)
}

func ReadUpdatePolicy() (currentVersion, debFile, updatePolicy string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}

	policyPath := filepath.Join(home, ".config", "fde_update.policy")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		return "", "", "", err
	}
	cfg, err := ini.Load(policyPath)
	if err != nil {
		return "", "", "", err
	}

	sec := cfg.Section(SectionUpdate)
	currentVersion = sec.Key(KeyCurrentVersion).String()
	debFile = sec.Key(KeyDebFile).String()
	updatePolicy = sec.Key(KeyUpdatePolicy).String()

	return currentVersion, debFile, updatePolicy, nil
}

const FDE_VERSION_UNINSTALLED = "uninstalled"

func VersionCurrentRead() (currentVersion string, err error) {
	propFile := "/var/lib/waydroid/waydroid.prop"
	_, err = os.Stat(propFile)
	if err != nil && os.IsNotExist(err) {
		return FDE_VERSION_UNINSTALLED, nil
	}
	data, err := os.ReadFile(propFile)
	if err != nil {
		logger.Warn("read_waydroid_prop_failed", err)
		return "", err
	} else {
		lines := string(data)
		for _, line := range strings.Split(lines, "\n") {
			if strings.HasPrefix(line, "ro.openfde.version=") {
				currentVersion = strings.TrimPrefix(line, "ro.openfde.version=")
				currentVersion = strings.TrimSpace(currentVersion)
				logger.Info("read_openfde_curr_version", currentVersion)
				break
			}
		}
	}
	return
}
