package windows_manager

import (
	"context"
	"fde_ctrl/logger"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
)

type WestonWM struct {
}

func getPrimaryDisplaySizes() (width, height int, err error) {
	display := os.Getenv("DISPLAY")
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		logger.Error("connect_xdisplay", nil, err)
		return
	}
	defer conn.Close()

	err = randr.Init(conn)
	if err != nil {
		logger.Error("init_randr", nil, err)
		return
	}

	root := xproto.Setup(conn).DefaultScreen(conn).Root
	primary, err := randr.GetOutputPrimary(conn, root).Reply()
	if err != nil {
		logger.Error("get_output_primary", nil, err)
		return
	}

	info, err := randr.GetOutputInfo(conn, primary.Output, 0).Reply()
	if err != nil {
		logger.Error("get_output_info", nil, err)
		return
	}

	if info.Crtc == 0 {
		logger.Error("crtc_is_0", nil, nil)
		return
	}

	crtc, err := randr.GetCrtcInfo(conn, info.Crtc, 0).Reply()
	if err != nil {
		logger.Error("get_crtc_info", nil, err)
		return
	}
	width = int(crtc.Width)
	height = int(crtc.Height)
	return
}

func getActivityDisplaySizes() (width, height string) {
	//run fde_display_geo.py to get the active display geometry

	output, err := exec.Command("python3", "fde_display_geo.py").Output()
	if err != nil {
		logger.Error("run_fde_display_geo", nil, err)
		return
	}
	outputStr := string(output)
	parts := strings.Split(outputStr, ",")
	if len(parts) != 2 {
		logger.Error("invalid_output_format", nil, nil)
		return
	}
	width = parts[0]
	height = parts[1]
	return
}

func (wm *WestonWM) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc) (cmdWinMan *exec.Cmd, err error) {
	var widthi, heighti int
	width, height := getActivityDisplaySizes()
	if width == "" || height == "" {
		logger.Error("get_activity_display_sizes", nil, nil)
		widthi, heighti, err = getPrimaryDisplaySizes()
		if err != nil {
			logger.Error("get_primary_display_sizes", nil, err)
			return
		}
		width = fmt.Sprint(widthi)
		height = fmt.Sprint(heighti)
	}

	cmdWeston := exec.CommandContext(mainCtx, "fde-weston", "--width="+width, "--height="+height, "--fullscreen")
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
