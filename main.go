// waybar-eyes based on face detection
package main

import (
	"fmt"
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

func main() {
	if len(os.Args) > 3 {
		fmt.Println("How to run:\n\t" + os.Args[0] + " [camera ID] [classifier XML file]")
		return
	}

	// get debug mode
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	boot := !debug

	// parse args
	// deviceID is my infrared built-in webcam
	deviceID := 0
	if len(os.Args) >= 2 {
		deviceID, _ = strconv.Atoi(os.Args[1])
	}

	// face model is the gocv default model example
	if len(os.Args) == 3 {
		XMLFile = os.Args[2]
	}

	// init waybar output
	var previousEyes eyes.Eyes

	// main loop here
	e := eyes.New(debug)

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
		faces, detected := detectFaceRepeat(deviceID, XMLFile, 10, debug)
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

func detectFaceRepeat(deviceID int, xmlFile string, repeat int, debug bool) (int, bool) {
	for i := 1; i <= repeat; i++ {
		if faces, detected := detectFace(deviceID, xmlFile, debug); detected {
			return faces, true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return 0, false
}

func detectFace(deviceID int, xmlFile string, debug bool) (int, bool) {
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

	return len(rects), len(rects) > 0

}
