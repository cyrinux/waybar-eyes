// waybar-eyes based on face detection
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cyrinux/waybar-eyes/helpers"
	"gocv.io/x/gocv"
)

// MaxEyes is the max number of eyes allowed
// in the waybar applet
const MaxEyes = 5

func main() {
	if len(os.Args) < 1 {
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
	var wOutput, previousWOutput helpers.WaybarOutput

	// main loop here
	count := 0
	eye := ""

	for {
		// detect face
		detected := detectFace(deviceID, xmlFile, debug)
		// increase or decrease eye counter
		// based on face detected or not
		if detected && count < MaxEyes {
			count++
		} else if count > 1 {
			count--
		}

		// print the eyes if eyes counter
		// positive
		wOutput.Class = "normal"
		if count == MaxEyes {
			wOutput.Class = "critical"
		}
		wOutput.Text = strings.Repeat(eye, count)
		wOutput.Tooltip = ""
		wOutput.Count = count

		// convert in JSON
		jsonOutput, err := json.Marshal(wOutput)
		if err != nil {
			fmt.Println(err)
		}

		// Finally print the expected waybar JSON
		if wOutput != previousWOutput {
			fmt.Println(string(jsonOutput))
		}
		previousWOutput = wOutput

		// sleep based on the eyes number
		// we want to quickly decrease the eyes
		// number if absence detected
		if !detected && count > 0 {
			time.Sleep(5 * time.Second)
		} else {
			// and the default sleep time
			time.Sleep(15 * time.Second)
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
	if len(rects) > 0 {
		if debug {
			fmt.Printf("found %d faces\n", len(rects))
		}
		return true
	}

	return false
}
