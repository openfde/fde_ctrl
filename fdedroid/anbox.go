package fdedroid

import (
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/tools"

	"context"
	"os"
	"os/exec"
	"time"
)

type Anbox struct {
}

const FDEDaemon = "fde_session"

func (fdedroid *Anbox) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, configure conf.Configure) (cmdFdeDaemon *exec.Cmd, err error) {
	// //step 2 stop kylin docker
	StopAndroidContainer(mainCtx, "kmre-1000-phytium")

	//step 3 start anbox hostside
	_, exist := tools.ProcessExists(FDEDaemon)
	if !exist {
		os.Remove("/tmp/anbox_started")
		//stop fdedroid
		err = StopAndroidContainer(mainCtx, FDEContainerName)
		if err != nil {
			logger.Error("start_fdedaemon_stop_fdedroid", nil, err)
			return
		}
		cmdFdeDaemon = exec.CommandContext(mainCtx, FDEDaemon, "session-manager", "--no-touch-emulation", "--single-window",
			"--window-size="+configure.Display.Resolution, "--standalone", "--experimental")
		cmdFdeDaemon.Env = append(os.Environ(), "LD_LIBRARY_PATH=/usr/local/fde/libs")
		err = cmdFdeDaemon.Start()
		if err != nil {
			logger.Error("start_fdedaemon", nil, err)
			return
		}
		go func() {
			err = cmdFdeDaemon.Wait()
			if err != nil {
				logger.Error("fde_session_wait", nil, err)
			}
			mainCtxCancelFunc()
		}()
		fileName := "/tmp/anbox_started"
		for i := 0; i < 3; i++ {
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				// 文件不存在，休眠 2 秒
				time.Sleep(2 * time.Second)
			} else {
				// 文件存在
				logger.Info("detected_file_exist", fileName)
				os.Remove(fileName)
				break
			}
		}
	}

	//step 4  start fde android container
	err = startAndroidContainer(mainCtx, configure.Android.Image, configure.Http.Host)
	if err != nil {
		logger.Error("start_android", nil, err)
		return
	}
	return
}
