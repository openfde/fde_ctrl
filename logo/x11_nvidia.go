package logo

// 本文件是 X11 splash 的 **NVIDIA 兼容实现**（新增文件，x11.go 的原实现保持不动）。
//
// 做法：窗口固定用 root visual（24 位、不透明、不建 colormap），像素按 visual 的
// mask/shift 自己打包，再用最原始的 xproto.PutImage 直写窗口。
//
// 为什么不复用 x11.go 那套：NVIDIA 专有驱动上，原来的 ARGB visual + XRender + MIT-SHM
// 链路经常出现「窗口建出来了、图却画不上」，屏幕上只剩一块黑底；这些失败点还是静默的
// （只写 Error 日志，窗口此时尚未 map），非常难查。
//

import (
	"encoding/binary"
	"errors"
	"fde_ctrl/logger"
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

// ---------- 像素打包 ----------

// scaleToMask 把 8bit 通道值缩放到 mask 对应的位宽（mask 必须是**未移位**的，
// 例如 0xff / 0x3ff）。
func scaleToMask(v uint8, mask uint32) uint32 {
	if mask == 0 {
		return 0
	}
	bits := uint(0)
	for m := mask; m != 0; m >>= 1 {
		bits++
	}
	if bits >= 8 {
		return uint32(v) << (bits - 8)
	}
	return uint32(v) >> (8 - bits)
}

// maskShift 返回 mask 最低有效位所在位置，也就是该通道需要的右移位数。
// 例如 0xff0000 → 16，0x0000ff → 0，0x3ff00000 → 20。
//
// 注意：xproto.VisualInfo 只给 mask，没有 shift 字段（那是 XRender 的 Directformat 才有）。
func maskShift(mask uint32) uint32 {
	if mask == 0 {
		return 0
	}
	shift := uint32(0)
	for mask&1 == 0 {
		mask >>= 1
		shift++
	}
	return shift
}

// packedPixel 按目标 visual 的 mask/shift 把颜色打包成一个像素值。
func packedPixel(c color.Color, rm, rs, gm, gs, bm, bs uint32) uint32 {
	r, g, b, _ := c.RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
	return (scaleToMask(r8, rm>>rs)<<rs)&rm |
		(scaleToMask(g8, gm>>gs)<<gs)&gm |
		(scaleToMask(b8, bm>>bs)<<bs)&bm
}

// buildImageBuffer 把整幅图按目标 visual 打包成可以直接 PutImage 的连续缓冲。
// 图比窗口小的时候，多出来的部分保持 0（黑色）。
func buildImageBuffer(img image.Image, w, h, bpp int, order byte,
	rm, rs, gm, gs, bm, bs uint32) []byte {
	buf := make([]byte, w*h*bpp)
	var enc binary.ByteOrder = binary.LittleEndian
	if order == xproto.ImageOrderMSBFirst {
		enc = binary.BigEndian
	}
	src := img.Bounds()
	cw, ch := src.Dx(), src.Dy()
	if cw > w {
		cw = w
	}
	if ch > h {
		ch = h
	}
	for y := 0; y < ch; y++ {
		row := buf[y*w*bpp:]
		for x := 0; x < cw; x++ {
			p := packedPixel(img.At(src.Min.X+x, src.Min.Y+y), rm, rs, gm, gs, bm, bs)
			switch bpp {
			case 4:
				enc.PutUint32(row[x*bpp:], p)
			case 2:
				enc.PutUint16(row[x*bpp:], uint16(p))
			}
		}
	}
	return buf
}

// ---------- 显示 ----------

func (logo *X11Logo) showWithPutImage() {
	// 检查当前环境是否为 X11
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "x11" {
		logger.Error("unsupported_session", nil, errors.New("XDG_SESSION_TYPE is not x11"))
		return
	}
	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("DISPLAY")
	if display == "" {
		return
	}

	X, err := xgb.NewConn()
	if err != nil {
		logger.Error("new_x_connection", nil, err)
		return
	}

	// 初始化 RandR（用于获取主显示器）
	randrSupported := false
	if err := randr.Init(X); err == nil {
		randrSupported = true
	} else {
		logger.Warn("randr_init_failed", err)
	}

	// 初始化 Xinerama（后备方案）
	xineramaSupported := false
	if err := xinerama.Init(X); err == nil {
		xineramaSupported = true
	} else {
		logger.Warn("xinerama_init_failed", err)
	}

	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	// 默认使用根窗口尺寸（单屏或降级）
	screenWidth := screen.WidthInPixels
	screenHeight := screen.HeightInPixels
	var screenX, screenY int16 = 0, 0

	// --- 优先使用 RandR 获取主显示器 ---
	if randrSupported {
		root := screen.Root
		resources, err := randr.GetScreenResources(X, root).Reply()
		if err != nil {
			logger.Warn("randr_get_screen_resources_failed", err)
		} else {
			primary, err := randr.GetOutputPrimary(X, root).Reply()
			if err != nil {
				logger.Warn("randr_get_output_primary_failed", err)
			} else {
				for _, output := range resources.Outputs {
					if output == primary.Output {
						oinfo, err := randr.GetOutputInfo(X, output, 0).Reply()
						if err != nil || oinfo.Crtc == 0 {
							continue
						}
						cinfo, err := randr.GetCrtcInfo(X, oinfo.Crtc, 0).Reply()
						if err != nil {
							continue
						}
						screenX = int16(cinfo.X)
						screenY = int16(cinfo.Y)
						screenWidth = uint16(cinfo.Width)
						screenHeight = uint16(cinfo.Height)
						break
					}
				}
			}
		}
	}

	// --- 如果 RandR 未能获取主显示器，则回退到 Xinerama 的第一个屏幕 ---
	if (screenWidth == 0 || screenHeight == 0) && xineramaSupported {
		reply, err := xinerama.QueryScreens(X).Reply()
		if err == nil && len(reply.ScreenInfo) >= 1 {
			target := reply.ScreenInfo[0]
			screenWidth = target.Width
			screenHeight = target.Height
			screenX = target.XOrg
			screenY = target.YOrg
			logger.Info("using_first_screen_from_xinerama", map[string]interface{}{
				"x": screenX, "y": screenY, "w": screenWidth, "h": screenHeight,
			})
		} else {
			logger.Warn("xinerama_query_failed", err)
		}
	}

	nvScreenW, nvScreenH = screenWidth, screenHeight

	img := CenterTileOpenFDE(int(screenWidth), int(screenHeight))
	if img == nil {
		// 背景图 /usr/share/backgrounds/openfde.png 打不开或解码失败时直接不建窗：
		// 否则会建出一块只有黑底的 splash 窗，挡在桌面上。
		logger.Error("logo_background_unavailable",
			"/usr/share/backgrounds/openfde.png", nil)
		X.Close()
		return
	}

	// 只用 root visual：depth 与屏幕一致、不透明、不需要额外 Colormap，
	// 像素直接 PutImage 就能写进去（splash 本来就是一张不透明全屏图）。
	visual, depth := screen.RootVisual, screen.RootDepth

	// 取 root visual 的像素布局（mask/shift），后面按它打包像素。
	var (
		rm, rs, gm, gs, bm, bs uint32
		haveVT                 bool
	)
	for _, d := range screen.AllowedDepths {
		for _, v := range d.Visuals {
			if v.VisualId == visual {
				rm, gm, bm = v.RedMask, v.GreenMask, v.BlueMask
				rs, gs, bs = maskShift(rm), maskShift(gm), maskShift(bm)
				haveVT = true
			}
		}
	}
	if !haveVT {
		logger.Error("no_visual_type", nil, nil)
		X.Close()
		return
	}
	if rm == 0 || gm == 0 || bm == 0 {
		logger.Error("bad_visual_masks", nil, nil)
		X.Close()
		return
	}

	// 每像素字节数由服务器给的 PixmapFormat 决定（depth 24 通常是 32bpp / 4 字节）。
	bpp := 0
	for _, pf := range setup.PixmapFormats {
		if pf.Depth == depth {
			bpp = int(pf.BitsPerPixel) / 8
		}
	}
	if bpp != 2 && bpp != 4 {
		logger.Error("unsupported_depth", nil, nil)
		X.Close()
		return
	}
	logger.Info("logo_visual", map[string]interface{}{
		"visual": visual, "depth": depth, "bpp": bpp,
	})

	wid, err := xproto.NewWindowId(X)
	if err != nil {
		logger.Error("new_window_id", nil, err)
		X.Close()
		return
	}

	windowWidth := uint16(screenWidth)
	windowHeight := uint16(screenHeight)
	windowX := screenX
	windowY := screenY

	// root visual 用服务器默认 colormap，不用自己建；背景填 0 即黑色。
	_ = xproto.CreateWindow(X, depth, wid, screen.Root,
		windowX, windowY, windowWidth, windowHeight, 0,
		xproto.WindowClassInputOutput, visual,
		xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwEventMask,
		[]uint32{0, 0,
			xproto.EventMaskStructureNotify | xproto.EventMaskExposure})

	// 整幅图先按 visual 打包成一个连续缓冲，后面 PutImage 直接发。
	pixbuf := buildImageBuffer(img, int(windowWidth), int(windowHeight), bpp,
		setup.ImageByteOrder, rm, rs, gm, gs, bm, bs)

	// 设置窗口类型为普通窗口
	wmWindowType := xproto.InternAtom(X, false, uint16(len("_NET_WM_WINDOW_TYPE")), "_NET_WM_WINDOW_TYPE")
	wmWindowTypeNormal := xproto.InternAtom(X, false, uint16(len("_NET_WM_WINDOW_TYPE_NORMAL")), "_NET_WM_WINDOW_TYPE_NORMAL")

	wmWindowTypeReply, _ := wmWindowType.Reply()
	wmWindowTypeNormalReply, _ := wmWindowTypeNormal.Reply()

	if wmWindowTypeReply != nil && wmWindowTypeNormalReply != nil {
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, wmWindowTypeReply.Atom,
			xproto.AtomAtom, 32, 1, (*[4]byte)(unsafe.Pointer(&wmWindowTypeNormalReply.Atom))[:])
	}

	// 准备窗口状态原子列表（用于 _NET_WM_STATE）
	wmState := xproto.InternAtom(X, false, uint16(len("_NET_WM_STATE")), "_NET_WM_STATE")
	wmStateReply, _ := wmState.Reply()

	var stateAtoms []uint32

	// 设置跳过任务栏
	wmStateSkipTaskbar := xproto.InternAtom(X, false, uint16(len("_NET_WM_STATE_SKIP_TASKBAR")), "_NET_WM_STATE_SKIP_TASKBAR")
	wmStateSkipTaskbarReply, _ := wmStateSkipTaskbar.Reply()
	if wmStateSkipTaskbarReply != nil {
		stateAtoms = append(stateAtoms, uint32(wmStateSkipTaskbarReply.Atom))
	}

	// 设置全屏
	wmStateFullscreen := xproto.InternAtom(X, false, uint16(len("_NET_WM_STATE_FULLSCREEN")), "_NET_WM_STATE_FULLSCREEN")
	wmStateFullscreenReply, _ := wmStateFullscreen.Reply()
	if wmStateFullscreenReply != nil {
		stateAtoms = append(stateAtoms, uint32(wmStateFullscreenReply.Atom))
	}

	if wmStateReply != nil && len(stateAtoms) > 0 {
		buf := make([]byte, 4*len(stateAtoms))
		for i, atom := range stateAtoms {
			xgb.Put32(buf[i*4:], atom)
		}
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, wmStateReply.Atom,
			xproto.AtomAtom, 32, uint32(len(stateAtoms)), buf)
	}

	// 去除窗口标题栏（_MOTIF_WM_HINTS）
	motifHintsAtom := xproto.InternAtom(X, false, uint16(len("_MOTIF_WM_HINTS")), "_MOTIF_WM_HINTS")
	motifHintsReply, _ := motifHintsAtom.Reply()
	if motifHintsReply != nil {
		// flags=2, functions=0, decorations=0, input_mode=0, status=0
		motifHints := []uint32{2, 0, 0, 0, 0}
		buf := make([]byte, 4*len(motifHints))
		for i, v := range motifHints {
			binary.LittleEndian.PutUint32(buf[i*4:], v)
		}
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, motifHintsReply.Atom,
			xproto.AtomCardinal, 32, uint32(len(motifHints)), buf)
	}

	// 设置窗口名称（也就是 wmctrl 里看到的 openfde.background）
	wmNameAtom := xproto.InternAtom(X, false, uint16(len("WM_NAME")), "WM_NAME")
	netWmNameAtom := xproto.InternAtom(X, false, uint16(len("_NET_WM_NAME")), "_NET_WM_NAME")
	utf8Atom := xproto.InternAtom(X, false, uint16(len("UTF8_STRING")), "UTF8_STRING")

	wmNameReply, _ := wmNameAtom.Reply()
	netWmNameReply, _ := netWmNameAtom.Reply()
	utf8Reply, _ := utf8Atom.Reply()

	title := []byte("openfde.background")
	if wmNameReply != nil && utf8Reply != nil {
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, wmNameReply.Atom,
			utf8Reply.Atom, 8, uint32(len(title)), title)
	}
	if netWmNameReply != nil && utf8Reply != nil {
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, netWmNameReply.Atom,
			utf8Reply.Atom, 8, uint32(len(title)), title)
	}

	// 注意：这里先不 MapWindow —— 等整幅图都写进窗口再映射，
	// 窗口一出现就是画好的样子，不会先闪一块纯黑底。

	gcid, err := xproto.NewGcontextId(X)
	if err != nil {
		logger.Error("new_gcontext_id", nil, err)
		xproto.DestroyWindow(X, wid)
		X.Close()
		return
	}
	_ = xproto.CreateGC(X, gcid, xproto.Drawable(wid), xproto.GcGraphicsExposures, []uint32{0})

	// 分条带 PutImage：单个请求不能太大（X 服务器对最大请求长有限制，
	// 1920x1080x4 一次发出去会被拒），这里限制在 240KB 以内。
	rowBytes := int(windowWidth) * bpp
	rowsPerPut := (240 * 1024) / rowBytes
	if rowsPerPut < 1 {
		rowsPerPut = 1
	}

	// xgb 里 PutImage 被声明为「不会返回错误」的请求，cookie 上调 Check() 会 panic，
	// 所以这里不 Check：真有 BadDrawable/BadGC 之类的问题，会在后面
	// PollForEvent 里以 xerr 的形式冒出来（我们只是刚建好的 drawable，基本不会）。
	drawAll := func() {
		for y := 0; y < int(windowHeight); y += rowsPerPut {
			rows := rowsPerPut
			if y+rows > int(windowHeight) {
				rows = int(windowHeight) - y
			}
			chunk := pixbuf[y*rowBytes : (y+rows)*rowBytes]
			xproto.PutImage(X, xproto.ImageFormatZPixmap, xproto.Drawable(wid),
				gcid, uint16(windowWidth), uint16(rows), 0, int16(y),
				0, depth, chunk)
		}
		X.Sync()
	}

	cleanup := func() {
		xproto.FreeGC(X, gcid)
		xproto.DestroyWindow(X, wid)
		X.Sync()
		X.Close()
	}

	drawAll()
	_ = xproto.MapWindow(X, wid)
	X.Sync()

	cancelCh := make(chan struct{})
	nvMu.Lock()
	nvCancelCh = cancelCh
	nvShowed = true
	nvDismissSched = false // 新一轮显示，延时排期重新开始
	pending := nvPendingDismiss
	nvPendingDismiss = false
	if pending {
		// Dismiss 在窗口出来之前就到了（Android 起得快 / 有别的收起来源先触发）。
		// 这时窗口刚画出来，同样让它露一会儿再收，不然等于什么都没显示。
		scheduleNvidiaDismissLocked(cancelCh)
	}
	nvMu.Unlock()

loop:
	for {
		ev, xerr := X.PollForEvent()
		if xerr != nil {
			logger.Error("x_event", nil, xerr)
			break
		}
		if ev == nil {
			select {
			case <-cancelCh:
				break loop
			default:
			}
			// xgb 没有「同时等多个条件」的接口，这里 10ms 轮询一次关闭信号
			time.Sleep(10 * time.Millisecond)
			continue
		}
		switch ev.(type) {
		case xproto.UnmapNotifyEvent:
			break loop
		case xproto.ExposeEvent:
			// 被遮挡/搬动后重画（用同一份缓冲，不碰 XRender）
			drawAll()
		}
	}

	cleanup()

	nvMu.Lock()
	nvCancelCh = nil
	nvMu.Unlock()
}

var (
	nvMu             sync.Mutex
	nvCancelCh       chan struct{} // 当前正在显示的 splash 的关闭信号
	nvShowed         bool
	nvPendingDismiss bool // 显示之前就收到的关闭请求，等显示出来后按延时兑现
	nvDismissSched   bool // 延时销毁是否已经排期，保证只排一次
	nvScreenW        uint16
	nvScreenH        uint16
)

// nvidiaDismissDelay 收到「收起」请求后延迟多久真正销毁窗口。
// 留 3 秒是为了让桌面 shell / Android 界面先画出来，splash 一撤不至于直接露黑底。
const nvidiaDismissDelay = 3 * time.Second

// scheduleNvidiaDismissLocked 安排「延时销毁窗口」，调用方必须持有 nvMu。
//
// 真正关窗由 Show 的轮询循环做（收到 cancelCh 关闭后 destroy），这里只是延时去
// close(cancelCh)。用 nvDismissSched 保证多个收起来源撞车时只排一次，
// 这样窗口存活时间恒定为「最后一次触发 + 3s」，而不会被反复顺延。
func scheduleNvidiaDismissLocked(cancelCh chan struct{}) {
	if nvDismissSched {
		return
	}
	nvDismissSched = true
	go func() {
		time.Sleep(nvidiaDismissDelay)
		nvMu.Lock()
		select {
		case <-cancelCh:
			// 已经关过了
		default:
			close(cancelCh)
		}
		nvMu.Unlock()
	}()
}

// dismissWithPutImage 会被多处调用（Android 启动完成的回调、以后可能的超时兜底等）。
// 这里必须幂等，也不能依赖调用时序：
//   - 窗口还没显示 → 先记一个 pending，等 Show 画完再排延时销毁；
//   - 已经显示 → 排一个 nvidiaDismissDelay（3s）的延时销毁
//     （xgb 的 Conn 不适合多协程并发写，真正的 DestroyWindow 仍由 Show 那个协程做）。
func (logo *X11Logo) dismissWithPutImage() {
	// 检查当前环境是否为 X11
	if os.Getenv("XDG_SESSION_TYPE") != "x11" {
		return
	}
	// 检查是否存在 DISPLAY 变量
	if os.Getenv("DISPLAY") == "" {
		return
	}

	nvMu.Lock()
	if !nvShowed {
		nvPendingDismiss = true
		nvMu.Unlock()
		return
	}
	cancelCh := nvCancelCh
	if cancelCh == nil {
		nvMu.Unlock()
		return // 窗口已经收掉了
	}
	select {
	case <-cancelCh:
		nvMu.Unlock()
		return // 已经有别的路径关过了
	default:
	}
	if nvDismissSched {
		nvMu.Unlock()
		return // 已经排期，等着就行
	}
	scheduleNvidiaDismissLocked(cancelCh)
	nvMu.Unlock()

	logger.Warn(fmt.Sprintf("screen size: %dx%d", nvScreenW, nvScreenH), nil)
	if nvScreenW <= 1920 {
		setDensity(160)
	} else {
		setDensity(256)
	}
}

