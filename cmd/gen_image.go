package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"os"

	"github.com/ojrac/opensimplex-go"
	"github.com/spf13/cobra"
	"hawx.me/code/img/blend"
)

var widthInt int
var heightInt int
var scaleInt int

func hslToRgb(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		gray := uint8(math.Round(l * 255))
		return gray, gray, gray
	}

	var temp1, temp2, huePercent, tempR, tempG, tempB float64
	if l < 0.5 {
		temp1 = l * (1 + s)
	} else {
		temp1 = l + s - l*s
	}

	temp2 = 2*l - temp1
	huePercent = h / 360

	tempR = constrain0to1(huePercent + 1.0/3.0)
	tempG = constrain0to1(huePercent)
	tempB = constrain0to1(huePercent - 1.0/3.0)

	convert := func(temp float64) float64 {
		if temp*6.0 < 1 {
			return temp2 + (temp1-temp2)*6.0*temp
		} else if temp*2.0 < 1 {
			return temp1
		} else if temp*3.0 < 2 {
			return temp2 + (temp1-temp2)*(2.0/3.0-temp)*6.0
		} else {
			return temp2
		}
	}

	toInt255 := func(temp float64) uint8 {
		return uint8(math.Round(temp * 255))
	}
	r = toInt255(convert(tempR))
	g = toInt255(convert(tempG))
	b = toInt255(convert(tempB))
	return r, g, b
}

func constrain0to1(value float64) float64 {
	if value < 0 {
		return constrain0to1(value + 1)
	}
	if value > 1 {
		return constrain0to1(value - 1)
	}
	return value
}

func pushToEdges(value float64, pad float64) float64 {
	pushRatio := 0.85
	if value < 0.5 {
		return constrain0to1(value*pushRatio + pad)
	} else {
		distanceTo1 := 1 - value
		return constrain0to1(1 - distanceTo1*pushRatio - pad)
	}
}

func createImg(nz1, nz2, nz3 opensimplex.Noise) image.Image {
	// Large scale makes the noise more smooth
	height := float64(heightInt)
	width := float64(widthInt)
	scale := float64(scaleInt)
	hueRange := float64(150)
	initialHue := rand.Float64() * 360
	lx := rand.Float64()*3 - 1.5
	ly := rand.Float64()*3 - 1.5

	// Create a new image
	img1 := image.NewRGBA(image.Rect(0, 0, widthInt, heightInt))

	// Set color for each pixel
	for y := 0; y < heightInt; y++ {
		for x := 0; x < widthInt; x++ {
			// Get noise value
			h := (initialHue + hueRange*nz3.Eval2(float64(x)/scale, float64(y)/scale) - (hueRange / 20))
			if h > 360 {
				h -= 360
			} else if h < 0 {
				h += 360
			}

			s := nz1.Eval2((lx*float64(x)+ly*float64(y))/scale, float64(y)/scale)
			s = 0.5 + s*0.5

			l := nz2.Eval2((lx*float64(x)+ly*float64(y))/scale, float64(y)/scale)
			l = pushToEdges(l, 0)
			r, g, b := hslToRgb(h, s, l)
			img1.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Create a new image
	img2 := image.NewRGBA(image.Rect(0, 0, widthInt, heightInt))

	// Set a new initial hue
	initialHue = rand.Float64() * 360

	// eff with x and y
	xOffset := int(math.Floor(rand.Float64()*width - width/2))
	yOffset := int(math.Floor(rand.Float64()*height - height/2))

	// Set color for each pixel
	for y := yOffset; y < heightInt+yOffset; y++ {
		for x := xOffset; x < widthInt+xOffset; x++ {
			// We want a sort of moving echo effect
			// Keep on x axis
			echo := nz3.Eval2(float64(x)/scale, float64(x)/scale)*(width/10) - (width / 20)
			// echo := 50.0

			// Get noise value
			h := (initialHue + hueRange*nz3.Eval2(float64(x)/scale, float64(y)/scale) - (hueRange / 20))
			if h > 360 {
				h -= 360
			} else if h < 0 {
				h += 360
			}
			// s := 0.5
			s := nz1.Eval2((lx*float64(x)+ly*float64(y)+echo)/scale, float64(y)/scale)
			s = 0.5 + s*0.5

			l := nz2.Eval2((lx*float64(x)+ly*float64(y)+echo)/scale, float64(y)/scale)
			l = pushToEdges(l, 0)
			r, g, b := hslToRgb(h, s, l)

			img2.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	img := blend.Dissolve(img1, img2)

	return img
}

var genImgCmd = &cobra.Command{
	Use:   "genimg",
	Short: "Generate a randomly painted image.",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		seed1 := rand.Int64()
		noise1 := opensimplex.NewNormalized(seed1)
		seed2 := rand.Int64()
		noise2 := opensimplex.NewNormalized(seed2)
		seed3 := rand.Int64()
		noise3 := opensimplex.NewNormalized(seed3)

		img1 := createImg(noise1, noise2, noise3)
		img2 := createImg(noise3, noise1, noise2)
		img3 := createImg(noise2, noise3, noise1)
		img := blend.Subtraction(img1, img2)
		img = blend.Dissolve(img, img3)

		// Save to out.png
		f, _ := os.Create("out.png")
		err := png.Encode(f, img)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	genImgCmd.Flags().IntVar(&widthInt, "width", 1800, "Width of the image")
	genImgCmd.Flags().IntVar(&heightInt, "height", 1200, "Height of the image")
	genImgCmd.Flags().IntVar(&scaleInt, "scale", 500, "Scale of the noise, higher is smoother")
	rootCmd.AddCommand(genImgCmd)
}
