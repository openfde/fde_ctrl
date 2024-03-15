package windows_manager

import (
	"context"
	"fde_ctrl/logger"
	"fmt"
	"os/exec"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type WestonWM struct {
}

func (wm *WestonWM) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc) (cmdWinMan *exec.Cmd, err error) {
	conn, err := xgb.NewConn()
	if err != nil {
		fmt.Printf("Failed to connect to X server: %v\n", err)
		return
	}
	defer conn.Close()

	screen := xproto.Setup(conn).DefaultScreen(conn)
	width := int(screen.WidthInPixels)
	height := int(screen.HeightInPixels)

	cmdWeston := exec.CommandContext(mainCtx, "fde-weston", "--width="+fmt.Sprint(width), "--height="+fmt.Sprint(height), "--fullscreen")
	err = cmdWeston.Start()
	if err != nil {
		logger.Error("start_weston", nil, err)
		mainCtxCancelFunc()
		return
	}
	go func() {
		err := cmdWeston.Wait()
		if err != nil {
			logger.Error("wait_weston", nil, err)
		}
		mainCtxCancelFunc()
	}()

	return cmdWeston, nil

}
