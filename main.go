// waybar-eyes based on face detection
// waybar config:
// ```
//
//	 "custom/eyes": {
//	  "exec": "cat ~/.cache/waybar-eyes.json",
//	  "interval": 5,
//	  "return-type": "json",
//	  "on-click": "pkill -f -SIGUSR1 waybar-eyes",
//	},
//
// ```
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gocv.io/x/gocv"
)

// MaxEyes is the max number of eyes allowed
// in the waybar applet
const SleepTimeOnPresence = 60 * time.Second
const SleepTimeOnAbsence = 30 * time.Second
const NewEyeTimeRate = 15 * time.Minute
const MaxEyes = 5
const EYE = ""

// Version give the software version
var Version string

// Eyes struct
type Eyes struct {
	WOuput WaybarOutput
	Count  int
}

// WaybarOutput struct
type WaybarOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
	Count   int    `json:"count"`
}

func main() {
	if len(os.Args) > 3 {
		fmt.Println("How to run:\n\tfacedetect [camera ID] [classifier XML file]")
		return
	}

	boot := true

	// get debug mode
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	// parse args
	// deviceID is my infrared built-in webcam
	deviceID := 0
	if len(os.Args) == 2 {
		deviceID, _ = strconv.Atoi(os.Args[1])
	}

	// face model is the gocv default model example
	xmlFile := "haarcascade_frontalface_default.xml"
	if len(os.Args) == 3 {
		xmlFile = os.Args[2]
	}

	// init waybar output
	var previousOutput WaybarOutput

	// main loop here
	var eyes Eyes

	// handle SIGUSR1 to reset count
	go eyes.signalHandler(debug)

	lastEyeTS := time.Now()
	for {
		// prevent poll on boot
		if boot {
			boot = false
			time.Sleep(SleepTimeOnPresence)
			continue
		}

		// increase based on face detected or not
		detected := detectFace5x(deviceID, xmlFile, debug)
		if detected && eyes.Count < MaxEyes && time.Since(lastEyeTS) > NewEyeTimeRate {
			eyes.Count++
			// keep timestamp of the last eye added
			lastEyeTS = time.Now()
		} else if !detected && eyes.Count > 0 {
			// decrease if no face detected
			// and count > 0
			eyes.Count--
		}

		// get formatted output
		eyes.getOutput()

		jsonOutput, err := json.Marshal(eyes.WOuput)
		if err != nil {
			continue
		}

		// Finally print the expected waybar JSON
		if debug {
			fmt.Println(string(jsonOutput))
		}

		// write the output in JSON cache file
		if eyes.WOuput != previousOutput {
			eyes.writeJSON(jsonOutput)
		}
		previousOutput = eyes.WOuput

		// sleep based on the eyes number
		// we want to quickly decrease the eyes
		// number if absence detected
		if !detected && eyes.Count > 0 {
			time.Sleep(SleepTimeOnAbsence)
		} else {
			// and the default sleep time
			time.Sleep(SleepTimeOnPresence)
		}
	}
}

func detectFace(deviceID int, xmlFile string, debug bool) bool {
	// open webcam
	webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	if err != nil {
		fmt.Println(err)
		return false
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
		return false
	}

	if ok := webcam.Read(&img); !ok {
		fmt.Printf("cannot read device %d\n", deviceID)
		return false
	}

	if img.Empty() {
		fmt.Printf("img empty %d\n", deviceID)
		return false
	}

	// detect faces
	rects := classifier.DetectMultiScale(img)
	if debug {
		fmt.Printf("found %d faces\n", len(rects))
	}

	return len(rects) > 0

}

func detectFace5x(deviceID int, xmlFile string, debug bool) bool {
	for i := 1; i <= 5; i++ {
		if detectFace(deviceID, xmlFile, debug) {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func (eyes *Eyes) writeJSON(output []byte) error {
	f, err := os.Create(os.Getenv("XDG_CACHE_HOME") + "/waybar-eyes.json")
	if err != nil {
		return err
	}
	f.WriteString(string(output))
	f.Close()

	return nil
}

func (eyes *Eyes) getOutput() {

	var output WaybarOutput
	output.Class = "normal"
	if eyes.Count == MaxEyes {
		output.Class = "critical"
	}
	output.Text = strings.Repeat(EYE, eyes.Count)
	output.Tooltip = ""
	output.Count = eyes.Count

	eyes.WOuput = output
}

func (eyes *Eyes) reset() {
	eyes.Count = 0
}

func (eyes *Eyes) signalHandler(debug bool) {
	// channel to trap signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGTERM, os.Interrupt)

	for {
		sig := <-sigs
		if sig == syscall.SIGUSR1 {
			eyes.reset()
			eyes.getOutput()

			jsonOutput, err := json.Marshal(eyes.WOuput)
			if err != nil {
				continue
			}

			if debug {
				fmt.Println(string(jsonOutput))
			}
			eyes.writeJSON(jsonOutput)
		} else {
			os.Exit(0)
		}
		time.Sleep(1 * time.Second)
	}
}
