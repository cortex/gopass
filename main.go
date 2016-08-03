package main

//go:generate genqrc assets/main.qml assets/logo.svg
import (
	"fmt"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"gopkg.in/qml.v1"
)

// UI is the model for the password UI
type UI struct {
	store    *PasswordStore
	hits     []Password
	query    string
	Len      int
	Selected int
	Status   string

	Countdown     float64
	countingDown  bool
	countdownDone chan bool

	ShowMetadata bool
	Metadata     string
	Info         string
	Cached       bool
}

// Quit the application
func (ui *UI) Quit() {
	os.Exit(0)
}

// Clearmetadata clears the displayed metadata
func (ui *UI) Clearmetadata() {
	ui.setMetadata("")
}

// ToggleShowMetadata toggles between showing and not showing metadata
func (ui *UI) ToggleShowMetadata() {
	ui.ShowMetadata = !ui.ShowMetadata
	qml.Changed(ui, &ui.ShowMetadata)
}

// Password gets the password at a specific index
func (ui *UI) Password(index int) Password {
	if index > len(ui.hits) {
		fmt.Println("Bad password fetch", index, len(ui.hits), ui.Len)
		return Password{}
	}
	pw := ui.hits[index]
	return pw
}

// ClearClipboard clears the clipboard
func (ui *UI) ClearClipboard() {
	if ui.countingDown {
		ui.countdownDone <- true
	}
	ui.countingDown = true
	tick := 10 * time.Millisecond
	t := time.NewTicker(tick)
	remaining := 15.0
	for {
		select {
		case <-ui.countdownDone:
			t.Stop()
			ui.countingDown = false
			return
		case <-t.C:
			ui.setCountdown(remaining)
			ui.setStatus(fmt.Sprintf("Will clear in %.f seconds", remaining))
			remaining -= tick.Seconds()
			if remaining <= 0 {
				clipboard.WriteAll("")
				ui.Clearmetadata()
				ui.setStatus("Clipboard cleared")
				ui.countingDown = false
				t.Stop()
				return
			}
		}
	}
}

// CopyToClipboard copies the selected password to the system clipboard
func (ui *UI) CopyToClipboard(selected int) {
	if selected >= len(ui.hits) {
		ui.setStatus("No password selected")
		return
	}
	pw := (ui.hits)[selected]
	pass := pw.Password()
	if err := clipboard.WriteAll(pass); err != nil {
		panic(err)
	}
	ui.setStatus("Copied to clipboard")
	go ui.ClearClipboard()
	ui.Update("") // Trigger a manual update, since the key is probably unlocked now
}

func (ui *UI) Select(selected int) {
	ui.Selected = selected
	// Trigger an update in a goroutine to keep QML from warning about a binding loop
	go func() { ui.Update("") }()
}

// Query updates the hitlist with the given query
func (ui *UI) Query(q string) {
	ui.query = q
	ui.setStatus(fmt.Sprintf("Matched %d items", ui.Len))
	ui.Update("queried")
}

func (ui *UI) setStatus(s string) {
	ui.Status = s
	qml.Changed(ui, &ui.Status)
}

func (ui *UI) setCountdown(c float64) {
	ui.Countdown = c
	qml.Changed(ui, &ui.Countdown)
}
func (ui *UI) setMetadata(s string) {
	ui.Metadata = s
	qml.Changed(ui, &ui.Metadata)
}

// Update is called whenever the store is updated, so the UI needs refreshing
func (ui *UI) Update(status string) {
	ui.hits = ui.store.Query(ui.query)
	ui.Len = len(ui.hits)
	var pw Password
	ui.Info = "Test"
	if ui.Selected < ui.Len {
		pw = (ui.hits)[ui.Selected]
		ki := pw.KeyInfo()
		ui.Info = fmt.Sprintf("Encrypted with %d bit %s key %s",
			ki.BitLength, ki.Algorithm, ki.Fingerprint)
		ui.Cached = ki.Cached
	}

	if ui.ShowMetadata {
		ui.Metadata = pw.Metadata()
	} else {
		ui.Metadata = "Press enter to decrypt"
	}

	qml.Changed(ui, &ui.Len)
	qml.Changed(ui, &ui.Info)
	qml.Changed(ui, &ui.Metadata)
	qml.Changed(ui, &ui.Cached)
	ui.setStatus(status)
}

var ui UI
var ps *PasswordStore

func main() {
	ps = NewPasswordStore()
	ui.store = ps
	ps.Subscribe(ui.Update)
	ui.Update("Started")
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	qml.SetApplicationName("GoPass\n")
	engine := qml.NewEngine()
	engine.Context().SetVar("passwords", &ui)
	controls, err := engine.LoadFile("qrc:/assets/main.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)
	window.Show()
	window.Wait()
	return nil
}
