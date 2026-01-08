package fdedroid

import (
	"context"
	"errors"
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/tools"
	"fde_ctrl/windows_manager"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type AppMode string

const APP_Fusing AppMode = "app_fusing"
const Desttop AppMode = "desktop"

type Waydroid struct {
}

func (fdedroid *Waydroid) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, conf conf.Configure, socket string, mode windows_manager.FDEMode) (cmdWaydroid *exec.Cmd, err error) {
	uid := os.Getuid()
	nativeFile := "/run/user/" + fmt.Sprint(uid) + "/pulse/native"
	if _, err := os.Stat(nativeFile); err != nil {
		if os.IsNotExist(err) {
			logger.Info("exist_pulse_native", "exist")
			_, exist := tools.ProcessExists("pulseaudio")
			if !exist {
				exec.Command("pulseaudio", "--start", "--log-target=journal").Start()
			}
		}
	}

	exec.Command("waydroid", "session", "stop").Run()
	// logger.Error("before waydroid_start", nil, nil)
	os.Environ()
	app_mode := ""
	confPath := "/etc/fde.d/fde.conf"
	if _, err := os.Stat(confPath); err == nil {
		content, err := os.ReadFile(confPath)
		if err == nil {
			lines := string(content)
			for _, line := range strings.Split(lines, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 && strings.TrimSpace(parts[0]) == "mode" {
					app_mode = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
	if app_mode == string(APP_Fusing) {
		if mode == windows_manager.DESKTOP_MODE_ENVIRONMENT {
			return nil, errors.New("app fusing mode is not supported in environment mode")
		}
		cmdWaydroid = exec.CommandContext(mainCtx, "waydroid", "session", "start")
	} else {
		cmdWaydroid = exec.CommandContext(mainCtx, "waydroid", "show-full-ui")
	}
	cmdWaydroid.Env = append(os.Environ(), "WAYLAND_DISPLAY="+socket)
	// logger.Error("before waydroid_start", "run", nil)
	// var stdout, stderr io.ReadCloser
	// stdout, err = cmdWaydroid.StdoutPipe()
	// if err != nil {
	// 	logger.Error("stdout pipe for vnc server", nil, err)
	// }
	// stderr, err = cmdWaydroid.StderrPipe()
	// if err != nil {
	// 	logger.Error("stderr pipe for vnc server", nil, err)
	// }
	err = cmdWaydroid.Start()
	if err != nil {
		logger.Error("run_waydroid", nil, err)
	}
	// logger.Error("err", stderr, nil)
	// logger.Error("out", stdout, nil)

	// output, err := ioutil.ReadAll(io.MultiReader(stdout, stderr))
	// if err != nil {
	// 	logger.Error("read start waydroid server failed", nil, err)
	// }
	go func() {
		err = cmdWaydroid.Wait()
		if err != nil {
			exec.Command("waydroid", "session", "stop").Run()
			logger.Error("waydroid_wait", nil, err)
		}
		mainCtxCancelFunc()
	}()
	return
}

func StopWaydroidContainer(ctx context.Context) error {
	cmdWaydroid := exec.CommandContext(ctx, "waydroid", "session", "stop")
	cmdWaydroid.Run()
	return nil
}
