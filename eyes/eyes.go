package eyes

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// EYE is a unicode eye
const EYE = ""

// MaxEyes is the max number of eyes allowed
// in the waybar applet
const MaxEyes = 5

// Eyes struct
type Eyes struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
	Count   int    `json:"count"`
	Debug   bool   `json:"-"`
}

// New eyes
func New(debug bool) Eyes {
	return Eyes{Debug: debug}
}

func (eyes *Eyes) PrepareWaybarOutput() {
	eyes.Class = "ok"
	if eyes.Count == MaxEyes {
		eyes.Class = "critical"
	}
	eyes.Text = strings.Repeat(EYE, eyes.Count)
	eyes.Tooltip = ""
}

func (eyes *Eyes) GetJSONOutput() []byte {
	jsonOutput, err := json.Marshal(eyes)
	if err != nil {
		return nil
	}

	// Finally print the expected waybar JSON
	if eyes.Debug {
		fmt.Println(string(jsonOutput))
	}

	return jsonOutput
}

func (eyes *Eyes) SignalHandler() {
	// channel to trap signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigs
		if sig == syscall.SIGUSR1 {
			eyes.Reset()
			eyes.PrepareWaybarOutput()
			json := eyes.GetJSONOutput()
			eyes.WriteJSONOutput(json)
		} else {
			os.Exit(0)
		}
		time.Sleep(1 * time.Second)
	}
}

func (eyes *Eyes) Reset() {
	eyes.Count = 0
}

func (eyes *Eyes) WriteJSONOutput(output []byte) error {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = os.Getenv("HOME") + "/.cache"
	}
	f, err := os.Create(cacheDir + "/waybar-eyes.json")
	if err != nil {
		return err
	}
	f.WriteString(string(output))
	f.Close()

	return nil
}
