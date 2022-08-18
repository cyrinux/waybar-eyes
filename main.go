// waybar-eyes based on face detection
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"time"

	"github.com/cyrinux/waybar-eyes/eyes"
	"gocv.io/x/gocv"
)

// SleepTimeOnPresence is the sleep time
// when a face is detected
const SleepTimeOnPresence = 60 * time.Second

// SleepTimeOnAbsence is the sleep time
// when no face is detected
const SleepTimeOnAbsence = 30 * time.Second

// NewEyeTimeRate is the time rate required
// to add a new eye in the output
const NewEyeTimeRate = 15 * time.Minute

// Version give the software version
var Version string

// XMLFile is the detection model use
var XMLFile = "haarcascade_frontalface_default.xml"

type Config struct {
	Debug  bool
	Device int
	Model  string
}

func main() {
	// config app
	config := Config{Debug: false, Device: 0, Model: XMLFile}
	flag.BoolVar(&config.Debug, "debug", config.Debug, "Debug mode")
	flag.IntVar(&config.Device, "d", config.Device, "Video device id, default: 0.")
	flag.StringVar(&config.Model, "m", config.Model, "Detection model path")
	flag.Parse()

	// get debug mode
	envDebug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if envDebug {
		config.Debug = envDebug
	}

	// If the app boot, we will skip the first loop
	// but if in debug, we will probe directly
	boot := !config.Debug

	// init waybar output
	var previousEyes eyes.Eyes

	// main loop here
	e := eyes.New(config.Debug)

	// handle SIGUSR1 to reset count
	go e.SignalHandler()

	lastEyeTS := time.Now()
	for {
		// prevent poll on boot
		if boot {
			boot = false
			time.Sleep(SleepTimeOnPresence)
			continue
		}

		// increase based on face detected or not
		faces, detected := detectFaceRepeat(config.Device, config.Model, 10, config.Debug)
		e.Faces = faces
		if detected && e.Count < eyes.MaxEyes && time.Since(lastEyeTS) > NewEyeTimeRate {
			e.Count++
			// keep timestamp of the last eye added
			lastEyeTS = time.Now()
		} else if !detected && e.Count > 0 {
			// decrease if no face detected
			// and count > 0
			e.Count--
		}

		// get formatted output
		e.PrepareWaybarOutput()
		jsonOutput := e.GetJSONOutput()

		// write the output in JSON cache file
		if e != previousEyes {
			e.WriteJSONOutput(jsonOutput)
		}
		previousEyes = e

		// sleep based on the eyes number
		// we want to quickly decrease the eyes
		// number if absence detected
		if !detected && e.Count > 0 {
			time.Sleep(SleepTimeOnAbsence)
		} else {
			// and the default sleep time
			time.Sleep(SleepTimeOnPresence)
		}
	}
}

// detectFaceRepeat try to detect a face several times
func detectFaceRepeat(deviceID int, xmlFile string, repeat int, debug bool) (int, bool) {
	for i := 1; i <= repeat; i++ {
		if faces, detected := detectFace(deviceID, xmlFile, debug); detected {
			return faces, true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return 0, false
}

// detectFace try to detect a face
func detectFace(deviceID int, xmlFile string, debug bool) (int, bool) {

	// debug
	window := gocv.NewWindow("Detect faces")
	defer window.Close()

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// open webcam
	webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	if err != nil {
		fmt.Println(err)
		return 0, false
	}
	defer webcam.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()
	if !classifier.Load(xmlFile) {
		fmt.Printf("Error reading cascade file: %v\n", xmlFile)
		return 0, false
	}

	if ok := webcam.Read(&img); !ok {
		fmt.Printf("cannot read device %d\n", deviceID)
		return 0, false
	}

	if img.Empty() {
		fmt.Printf("img empty %d\n", deviceID)
		return 0, false
	}

	// detect faces
	rects := classifier.DetectMultiScale(img)

	if debug {
		// draw a rectangle around each face on the original image,
		// along with text identifying as "Human"
		for _, r := range rects {
			gocv.Rectangle(&img, r, blue, 3)

			size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		// show the image in the window, and wait 1 millisecond
		window.IMShow(img)
		window.WaitKey(500)
	}

	return len(rects), len(rects) > 0

}
