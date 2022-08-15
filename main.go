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
const SleepTimeOnPresence = 30 * time.Second
const SleepTimeOnAbsence = 15 * time.Second
const NewEyeTimeRate = 1 * time.Minute
const MaxEyes = 5
const EYE = ""

// Version give the software version
var Version string

// Eyes struct
type Eyes struct {
	Count int
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

	// get debug mode
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	// parse args
	// deviceID is my infrared built-in webcam
	deviceID := 2
	if len(os.Args) == 2 {
		deviceID, _ = strconv.Atoi(os.Args[1])
	}

	// face model is the gocv default model example
	xmlFile := "haarcascade_frontalface_default.xml"
	if len(os.Args) == 3 {
		xmlFile = os.Args[2]
	}

	// init waybar output
	var wOutput, previousWOutput WaybarOutput

	// main loop here
	var eyes Eyes
	go eyes.signalHandler(debug)

	lastEyeTS := time.Now()
	for {
		// detect face
		detected := detectFace(deviceID, xmlFile, debug)
		// increase or decrease eye counter
		// based on face detected or not
		if detected && (time.Since(lastEyeTS)) > NewEyeTimeRate && eyes.Count < MaxEyes {
			eyes.Count++
			lastEyeTS = time.Now()
		} else if eyes.Count > 0 {
			eyes.Count--
		}

		// get formatted output
		output, _ := eyes.getOutput()

		// Finally print the expected waybar JSON
		if debug {
			fmt.Println(string(output))
		}

		// write the output in JSON cache file
		if wOutput != previousWOutput {
			eyes.writeJSON(output)
		}
		previousWOutput = wOutput

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
		return false
	}

	// detect faces
	rects := classifier.DetectMultiScale(img)
	return len(rects) > 0

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

func (eyes *Eyes) getOutput() ([]byte, error) {
	var wOutput WaybarOutput

	wOutput.Class = "normal"
	if eyes.Count == MaxEyes {
		wOutput.Class = "critical"
	}
	wOutput.Text = strings.Repeat(EYE, eyes.Count)
	wOutput.Tooltip = ""
	wOutput.Count = eyes.Count

	jsonOutput, err := json.Marshal(wOutput)
	if err != nil {
		return nil, err
	}

	return jsonOutput, nil
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
			output, _ := eyes.getOutput()
			if debug {
				fmt.Println(string(output))
			}
			eyes.writeJSON(output)
		} else {
			os.Exit(0)
		}
		time.Sleep(1 * time.Second)
	}
}
