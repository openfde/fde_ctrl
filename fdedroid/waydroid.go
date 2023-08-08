package fdedroid

import (
	"context"
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"os/exec"
)

type Waydroid struct {
}

func (fdedroid *Waydroid) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, conf conf.Configure) (cmdWaydroid *exec.Cmd, err error) {
	// logger.Error("before waydroid_start", nil, nil)
	cmdWaydroid = exec.CommandContext(mainCtx, "waydroid", "show-full-ui")
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
