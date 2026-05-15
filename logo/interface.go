package logo

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"strconv"
	"fde_ctrl/logger"
	"os/exec"


	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Logo interface {
	// SetUpgrading()
	Show()
	Dismiss()
}





func ShowLogo(xdgSessionType string) {
	var logo Logo
	if xdgSessionType == "wayland" {
		logo = &WaylandLogo{}
	} else {
		logo = &X11Logo{}
	}
	logo.Show()
}

func DismissLogo(xdgSessionType string) {
	var logo Logo
	if xdgSessionType == "wayland" {
		logo = &WaylandLogo{}
	} else {
		logo = &X11Logo{}
	}
	logo.Dismiss()
}

func setDensity(density int) {
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

const ENV_OPENFDE_UPGRADING = "OPENFDE_UPGRADING"

func SetUpgrading() {
	os.Setenv(ENV_OPENFDE_UPGRADING, "1")
}


// CenterTileOpenFDE draws the OpenFDE logo centered on a screen-sized background.
func CenterTileOpenFDE(screenWidth, screenHeight int) (image.Image) {
		// Load a picture from the command line.
	f, err := os.Open("/usr/share/backgrounds/openfde.png")
	if err != nil {
		logger.Error("open_image", nil, err)
		return nil
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		logger.Error("decode_image", nil, err)
		return nil
	}
	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	result := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	var sRGBBackgroundOfLogo color.RGBA = color.RGBA{61, 60, 54, 255}
	draw.Draw(result, result.Bounds(), &image.Uniform{sRGBBackgroundOfLogo}, image.Point{}, draw.Src)

	offsetX := (screenWidth - imgWidth) / 2
	offsetY := (screenHeight - imgHeight) / 2

	dstRect := image.Rect(offsetX, offsetY, offsetX+imgWidth, offsetY+imgHeight)
	//裁剪img到dstRect大小，并绘制到result上
	draw.Draw(result, dstRect, img, img.Bounds().Min, draw.Over)

	if os.Getenv(ENV_OPENFDE_UPGRADING) == "1" {
		fontSize := float64(screenHeight) * 0.06
		if fontSize < 56 {
			fontSize = 56
		}
		result = DrawInstallingText(result, "Upgrading", fontSize)
	}

	return result
}

// DrawInstallingText overlays centered bottom text with a subtle shadow.
func DrawInstallingText(src image.Image, text string, fontSize float64) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)

	candidates := []string{
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJKsc-Regular.otf",
	}

	var face font.Face
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		ft, err := opentype.Parse(data)
		if err != nil {
			continue
		}
		f, err := opentype.NewFace(ft, &opentype.FaceOptions{
			Size:    fontSize,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err == nil {
			face = f
			break
		}
	}
	if face == nil {
		if ft, err := opentype.Parse(goregular.TTF); err == nil {
			if f, err := opentype.NewFace(ft, &opentype.FaceOptions{
				Size:    fontSize,
				DPI:     72,
				Hinting: font.HintingFull,
			}); err == nil {
				face = f
			}
		}
	}
	if face == nil {
		face = basicfont.Face7x13
	}

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{255, 255, 255, 255}),
		Face: face,
	}

	w := d.MeasureString(text).Round()
	x := (b.Dx() - w) / 2
	y := b.Dy() - int(fontSize*1.4)
	if y < int(fontSize) {
		y = b.Dy() / 2
	}

	shadow := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{0, 0, 0, 180}),
		Face: face,
		Dot:  fixed.P(x+2, y+2),
	}
	shadow.DrawString(text)

	d.Dot = fixed.P(x, y)
	d.DrawString(text)

	return dst
}