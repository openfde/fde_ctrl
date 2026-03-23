package logo

import (
	"encoding/binary"
	"errors"
	"fde_ctrl/logger"
	"os"
	"os/exec"
	"strconv"
	"reflect"
	"unsafe"
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"

	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// #include <sys/ipc.h>
// #include <sys/shm.h>
import "C"

func F64ToFixed(f float64) render.Fixed { return render.Fixed(f * 65536) }
func FixedToF64(f render.Fixed) float64 { return float64(f) / 65536 }

func setDensity(density int){
	cmd := exec.Command("fde_fs", "-density", strconv.Itoa(density))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("set density failed", map[string]interface{}{
			"density": density,
			"output":  string(output),
		}, err)
	} else {
	    logger.Warn("set density %d success\n", density)
	}
}

var formats = map[byte]struct {
	format    render.Directformat
	transform func(color.Color) uint32
}{
	32: {
		format: render.Directformat{
			RedShift:   16,
			RedMask:    0xff,
			GreenShift: 8,
			GreenMask:  0xff,
			BlueShift:  0,
			BlueMask:   0xff,
			AlphaShift: 24,
			AlphaMask:  0xff,
		},
		transform: func(color color.Color) uint32 {
			r, g, b, a := color.RGBA()
			return (a>>8)<<24 | (r>>8)<<16 | (g>>8)<<8 | (b >> 8)
		},
	},
	30: {
		/*
			// Alpha makes compositing unbearably slow.
			AlphaShift: 30,
			AlphaMask:  0x3,
		*/
		format: render.Directformat{
			RedShift:   20,
			RedMask:    0x3ff,
			GreenShift: 10,
			GreenMask:  0x3ff,
			BlueShift:  0,
			BlueMask:   0x3ff,
		},
		transform: func(color color.Color) uint32 {
			r, g, b, a := color.RGBA()
			return (a>>14)<<30 | (r>>6)<<20 | (g>>6)<<10 | (b >> 6)
		},
	},
}

func Show() {
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

	// Load a picture from the command line.
	f, err := os.Open("/usr/share/backgrounds/openfde.png")
	if err != nil {
		logger.Error("open_image", nil, err)
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		logger.Error("decode_image", nil, err)
		return
	}

	// Miscellaneous X11 initialization.
	X, err := xgb.NewConn()
	if err != nil {
		logger.Error("new_x_connection", nil, err)
		return
	}

	if err := render.Init(X); err != nil {
		logger.Error("init_render", nil, err)
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
						logger.Info("using_primary_screen_from_randr", map[string]interface{}{
							"x": screenX, "y": screenY, "w": screenWidth, "h": screenHeight,
						})
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
			target := reply.ScreenInfo[0] // 第一个屏幕
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

	// 如果仍然没有有效尺寸，保留根窗口尺寸（已在上面设置）
	// 此时 screenX, screenY 均为 0
	screenWidthGlobal, screenHeightGlobal = screenWidth, screenHeight

	var sRGBBackgroundOfLogo color.RGBA = color.RGBA{61, 60, 54, 255}
	img = CenterTileImage(img, int(screenWidth), int(screenHeight), sRGBBackgroundOfLogo)

	visual, depth := screen.RootVisual, screen.RootDepth

	// Only go for 10-bit when the picture can make use of that range.
	prefer30 := false
	switch img.(type) {
	case *image.Gray16, *image.RGBA64, *image.NRGBA64:
		prefer30 = true
	}

	// XXX: We don't /need/ alpha here, it's just a minor improvement--affects
	// the backpixel value. (And we reject it in 30-bit depth anyway.)
Depths:
	for _, i := range screen.AllowedDepths {
		for _, v := range i.Visuals {
			// TODO: Could/should check other parameters, e.g., the RGB masks.
			if v.Class != xproto.VisualClassTrueColor {
				continue
			}
			if i.Depth == 32 || (i.Depth == 30 && prefer30) {
				visual, depth = v.VisualId, i.Depth
				if !prefer30 || i.Depth == 30 {
					break Depths
				}
			}
		}
	}

	format, ok := formats[depth]
	if !ok {
		logger.Error("unsupported_depth", nil, nil)
		return

	}

	mid, err := xproto.NewColormapId(X)
	if err != nil {
		logger.Error("new_colormap_id", nil, err)
		return
	}

	_ = xproto.CreateColormap(
		X, xproto.ColormapAllocNone, mid, screen.Root, visual)

	wid, err := xproto.NewWindowId(X)
	if err != nil {
		logger.Error("new_window_id", nil, err)
		return
	}

	windowWidth := uint16(screenWidth)
	windowHeight := uint16(screenHeight)
	windowX := screenX
	windowY := screenY

	// 背景色改为不透明黑色
	backPixel := format.transform(color.RGBA{0, 0, 0, 255})

	_ = xproto.CreateWindow(X, depth, wid, screen.Root,
		windowX, windowY, windowWidth, windowHeight, 0,
		xproto.WindowClassInputOutput, visual,
		xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwEventMask|xproto.CwColormap,
		[]uint32{backPixel, 0,
			xproto.EventMaskStructureNotify | xproto.EventMaskExposure,
			uint32(mid)})

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
		// Convert []uint32 to []byte for ChangeProperty
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
		// 结构体定义见 https://specifications.freedesktop.org/wm-spec/wm-spec-latest.html#idm45841325324160
		// flags=2, functions=0, decorations=0, input_mode=0, status=0
		motifHints := []uint32{2, 0, 0, 0, 0}
		buf := make([]byte, 4*len(motifHints))
		for i, v := range motifHints {
			binary.LittleEndian.PutUint32(buf[i*4:], v)
		}
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, motifHintsReply.Atom,
			xproto.AtomCardinal, 32, uint32(len(motifHints)), buf)
	}

	// 设置窗口名称
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

	_ = xproto.MapWindow(X, wid)

	pformats, err := render.QueryPictFormats(X).Reply()
	if err != nil {
		logger.Error("query_pict_formats", nil, err)
		return
	}

	// 查找与 visual 匹配的 PictFormat
	var pformat render.Pictformat
	for _, pd := range pformats.Screens[X.DefaultScreen].Depths {
		// This check seems to be slightly extraneous.
		if pd.Depth != depth {
			continue
		}
		for _, pv := range pd.Visuals {
			if pv.Visual == visual {
				pformat = pv.Format
			}
		}
	}

	// Wrap the window's surface in a picture.
	pid, err := render.NewPictureId(X)
	if err != nil {
		logger.Error("new_picture_id", nil, err)
		return
	}
	render.CreatePicture(X, pid, xproto.Drawable(wid), pformat, 0, []uint32{})

	// setup.BitmapFormatScanline{Pad,Unit} and setup.BitmapFormatBitOrder
	// don't interest us here since we're only using Z format pixmaps.
	for _, pf := range setup.PixmapFormats {
		if pf.Depth == depth {
			if pf.BitsPerPixel != 32 || pf.ScanlinePad != 32 {
				logger.Error("unsuported X server", nil, errors.New("bitsperpixel or scanlinepad not supported"))
				return
			}
		}
	}

	pixid, err := xproto.NewPixmapId(X)
	if err != nil {
		logger.Error("new_pixmap_id", nil, err)
		return
	}
	_ = xproto.CreatePixmap(X, depth, pixid, xproto.Drawable(screen.Root),
		uint16(img.Bounds().Dx()), uint16(img.Bounds().Dy()))

	var bgraFormat render.Pictformat
	for _, pf := range pformats.Formats {
		if pf.Depth == depth && pf.Direct == format.format {
			bgraFormat = pf.Id
			break
		}
	}
	if bgraFormat == 0 {
		logger.Error("pictformat_not_found", nil, nil)
		return
	}

	// We could also look for the inverse pictformat.
	var encoding binary.ByteOrder
	if setup.ImageByteOrder == xproto.ImageOrderMSBFirst {
		encoding = binary.BigEndian
	} else {
		encoding = binary.LittleEndian
	}

	pixpicid, err := render.NewPictureId(X)
	if err != nil {
		logger.Error("new_picture_id", nil, err)
		return
	}
	render.CreatePicture(X, pixpicid, xproto.Drawable(pixid), bgraFormat,
		0, []uint32{})

	// Do we really need this? :/
	cid, err := xproto.NewGcontextId(X)
	if err != nil {
		logger.Error("new_gcontext_id", nil, err)
		return
	}
	_ = xproto.CreateGC(X, cid, xproto.Drawable(pixid),
		xproto.GcGraphicsExposures, []uint32{0})

	bounds := img.Bounds()

	if err := shm.Init(X); err != nil {
		logger.Error("init_mit_shm", nil, err)
		return
		// We're being lazy and resolve the 1<<16 limit of requests by sending
		// a row at a time. The encoding is also done inefficiently.
		// Also see xgbutil/xgraphics/xsurface.go.
		row := make([]byte, bounds.Dx()*4)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				encoding.PutUint32(row[x*4:], format.transform(img.At(x, y)))
			}
			_ = xproto.PutImage(X, xproto.ImageFormatZPixmap,
				xproto.Drawable(pixid), cid, uint16(bounds.Dx()), 1,
				0, int16(y),
				0, depth, row)
		}
	} else {
		rep, err := shm.QueryVersion(X).Reply()
		if err != nil {
			logger.Error("query_version", nil, err)
			return
		}
		if rep.PixmapFormat != xproto.ImageFormatZPixmap ||
			!rep.SharedPixmaps {
			logger.Error("mit_shm_unfit_failed", nil, nil)
			return
		}

		shmSize := bounds.Dx() * bounds.Dy() * 4

		// As a side note, to clean up unreferenced segments (orphans):
		//  ipcs -m | awk '$6 == "0" { print $2 }' | xargs ipcrm shm
		shmID := int(C.shmget(C.IPC_PRIVATE,
			C.size_t(shmSize), C.IPC_CREAT|0777))
		if shmID == -1 {
			// TODO: We should handle this case by falling back to PutImage,
			// if only because the allocation may hit a system limit.
			logger.Error("shmget", nil, errors.New("shmget failed"))
			return
		}

		dataRaw := C.shmat(C.int(shmID), nil, 0)
		defer C.shmdt(dataRaw)
		defer C.shmctl(C.int(shmID), C.IPC_RMID, nil)

		data := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(dataRaw), Len: shmSize, Cap: shmSize}))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			row := data[y*bounds.Dx()*4:]
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				encoding.PutUint32(row[x*4:], format.transform(img.At(x, y)))
			}
		}

		segid, err := shm.NewSegId(X)
		if err != nil {
			logger.Error("new_segid", nil, err)
			return
		}

		// Need to have it attached on the server before we unload the segment.
		c := shm.AttachChecked(X, segid, uint32(shmID), true /* RO */)
		if err := c.Check(); err != nil {
			logger.Error("shm_attach", nil, err)
			return
		}

		_ = shm.PutImage(X, xproto.Drawable(pixid), cid,
			uint16(bounds.Dx()), uint16(bounds.Dy()), 0, 0,
			uint16(bounds.Dx()), uint16(bounds.Dy()), 0, 0,
			depth, xproto.ImageFormatZPixmap,
			0 /* SendEvent */, segid, 0 /* Offset */)
	}

	// 用于存储窗口当前尺寸（缓存，在 Expose 中也会实时获取）
	var winWidth, winHeight uint16 = windowWidth, windowHeight

	go func() {
		logoShowedx11 = true
		defer close(done)
		select {
		case <-done:
			// Clean up X11 resources
			render.FreePicture(X, pid)
			render.FreePicture(X, pixpicid)
			xproto.FreePixmap(X, pixid)
			xproto.FreeGC(X, cid)
			xproto.FreeColormap(X, mid)
			xproto.DestroyWindow(X, wid)
			X.Close()
		}
	}()

	for {
		ev, xerr := X.WaitForEvent()
		if xerr != nil {
			logger.Error("x_event", nil, xerr)
			return
		}
		if ev == nil {
			return
		}

		//log.Printf("Event: %s\n", ev)
		switch e := ev.(type) {
		case xproto.UnmapNotifyEvent:
			return

		case xproto.ConfigureNotifyEvent:
			// 更新窗口尺寸
			winWidth, winHeight = e.Width, e.Height

			// 计算缩放比例，使图片等比例适应窗口（可能裁剪）
			scaleX := float64(bounds.Dx()) / float64(winWidth)
			scaleY := float64(bounds.Dy()) / float64(winHeight)
			var scale float64
			if scaleX < scaleY {
				scale = scaleY
			} else {
				scale = scaleX
			}

			_ = render.SetPictureTransform(X, pixpicid, render.Transform{
				F64ToFixed(scale), 0, 0,
				0, F64ToFixed(scale), 0,
				0, 0, F64ToFixed(1),
			})
			_ = render.SetPictureFilter(X, pixpicid, 8, "bilinear", nil)

		case xproto.ExposeEvent:
			// 实时获取窗口当前尺寸（确保绘制覆盖全窗口）
			geom, err := xproto.GetGeometry(X, xproto.Drawable(wid)).Reply()
			if err != nil {
				logger.Warn("get_geometry_failed", err)
				// 失败时回退到缓存的尺寸
			} else {
				winWidth, winHeight = geom.Width, geom.Height
			}

			_ = render.Composite(X, render.PictOpSrc,
				pixpicid, render.PictureNone, pid,
				0, 0, 0, 0, 0, 0,
				winWidth, winHeight)
		}
	}
}

var done = make(chan struct{})
var logoShowedx11 = false
var screenWidthGlobal   uint16
var screenHeightGlobal  uint16

func Disappear() {
	if logoShowedx11 == false {
		return
	}
	logger.Warn(fmt.Sprintf("screen size: %dx%d", screenWidthGlobal, screenHeightGlobal), nil)
	if screenWidthGlobal <= 1920 {
		setDensity(160)
	} else {
		setDensity(256)
	}
	// 检查当前环境是否为 X11
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "x11" {
		return
	}

	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("DISPLAY")
	if display == "" {
		return
	}
	// 发送关闭信号
	done <- struct{}{}
}

// 改为接受背景色并使用 alpha 混合（draw.Over）
func CenterTileImage(img image.Image, screenWidth, screenHeight int, bg color.Color) image.Image {
	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	// Create a new RGBA image with the size of the screen
	result := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))

	// Fill the background with bg color (opaque or with its own alpha)
	draw.Draw(result, result.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Calculate the top-left position to center the image
	offsetX := (screenWidth - imgWidth) / 2
	offsetY := (screenHeight - imgHeight) / 2

	// Draw the image at the center using Over to respect the source alpha
	dstRect := image.Rect(offsetX, offsetY, offsetX+imgWidth, offsetY+imgHeight)
	draw.Draw(result, dstRect, img, img.Bounds().Min, draw.Over)

	return result
}
