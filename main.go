package main

//go:generate go run vendor/github.com/limetext/qml-go/cmd/genqrc/main.go assets
import (
	"fmt"
	"os"
	"time"

	"sync"

	"github.com/atotto/clipboard"
	"github.com/limetext/qml-go"
)

const (
	timeoutTickDuration = 10 * time.Millisecond
	clipboardTimeout    = 15 * time.Second
)

// UI is the model for the password UI
type UI struct {
	Status string
	query  string

	Countdown float64
	counter   Counter

	ShowMetadata bool

	Password struct {
		Name     string
		Metadata string
		Info     string
		Cached   bool
	}
}

type Counter struct {
	sync.Mutex
	t            *time.Ticker
	remaining    time.Duration
	countingDown bool
}

func (c *Counter) isRunning() bool {
	c.Lock()
	defer c.Unlock()
	return c.countingDown
}

func (c *Counter) start(onTick func(remaining float64)) {
	c.Lock()
	defer c.Unlock()
	c.remaining = clipboardTimeout
	c.countingDown = true
	c.t = time.NewTicker(timeoutTickDuration)
	go func() {
		for {
			<-c.t.C
			c.Lock()
			c.remaining -= timeoutTickDuration
			c.Unlock()
			onTick(c.remaining.Seconds())
		}
	}()
}

func (c *Counter) stop() {
	c.Lock()
	defer c.Unlock()
	c.countingDown = false
	c.t.Stop()
}

func (c *Counter) reset() {
	c.Lock()
	defer c.Unlock()
	c.remaining = clipboardTimeout
}

// Passwords is the model for the password list
type Passwords struct {
	Selected int
	Len      int
	store    *PasswordStore
	hits     []Password
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
	passwords.Update("")
	qml.Changed(ui, &ui.ShowMetadata)
}

// Get gets the password at a specific index
func (p *Passwords) Get(index int) Password {
	if index > len(p.hits) {
		fmt.Println("Bad password fetch", index, len(p.hits), p.Len)
		return Password{}
	}
	pw := p.hits[index]
	return pw
}

// ClearClipboard clears the clipboard
func (ui *UI) ClearClipboard() {

	if ui.counter.isRunning() {
		ui.counter.reset()
		return
	}

	onTick := func(remaining float64) {
		ui.setCountdown(ui.counter.remaining.Seconds())
		ui.setStatus(fmt.Sprintf("Will clear in %.f seconds", remaining))
		if remaining <= 0 {
			clipboard.WriteAll("")
			ui.Clearmetadata()
			ui.setStatus("Clipboard cleared")
			ui.counter.stop()
		}
	}

	ui.counter.start(onTick)
}

// CopyToClipboard copies the selected password to the system clipboard
func (p *Passwords) CopyToClipboard(selected int) {
	if selected >= len(p.hits) {
		ui.setStatus("No password selected")
		return
	}
	pw := (p.hits)[selected]
	pass, err := pw.Password()
	if err != nil {
		ui.setStatus("Cancelled")
		return
	}

	if err := clipboard.WriteAll(pass); err != nil {
		panic(err)
	}
	ui.setStatus("Copied to clipboard")
	ui.ClearClipboard()
	p.Update("") // Trigger a manual update, since the key is probably unlocked now
}

// Select the password with the specified index
func (p *Passwords) Select(selected int) {
	p.Selected = selected
	// Trigger an update in a goroutine to keep QML from warning about a binding loop
	go func() { p.Update("") }()
}

// Query updates the hitlist with the given query
func (ui *UI) Query(q string) {
	ui.query = q
	passwords.Update("queried")
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
	ui.Password.Metadata = s
	qml.Changed(ui, &ui.Password.Metadata)
}

// Update is called whenever the store is updated, so the UI needs refreshing
func (p *Passwords) Update(status string) {
	p.hits = p.store.Query(ui.query)
	p.Len = len(p.hits)

	var pw Password

	ui.Password.Info = "Test"
	if p.Selected < p.Len {
		pw = (p.hits)[p.Selected]
		ki := pw.KeyInfo()
		if ki.Algorithm != "" {
			ui.Password.Info = fmt.Sprintf("Encrypted with %d bit %s key %s",
				ki.BitLength, ki.Algorithm, ki.Fingerprint)
			ui.Password.Cached = ki.Cached
		} else {
			ui.Password.Info = "Not encrypted"
			ui.Password.Cached = false
		}
		ui.Password.Name = pw.Name
	}

	if ui.ShowMetadata {
		ui.Password.Metadata = pw.Metadata()
	} else {
		ui.Password.Metadata = "Press enter to decrypt"
		ui.Password.Metadata = pw.Raw()
	}
	qml.Changed(p, &p.Len)
	qml.Changed(&ui, &ui.Password)
	qml.Changed(&ui, &ui.Password.Metadata)
	qml.Changed(&ui, &ui.Password.Name)
	ui.setStatus(status)
}

var ui UI
var passwords Passwords
var ps *PasswordStore

func main() {
	ps = NewPasswordStore()
	passwords.store = ps
	ps.Subscribe(passwords.Update)
	passwords.Update("Started")
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	qml.SetApplicationName("GoPass")
	engine := qml.NewEngine()
	engine.Context().SetVar("passwords", &passwords)
	engine.Context().SetVar("ui", &ui)
	_, err := engine.LoadFile("qrc:/assets/RoundButton.qml")
	if err != nil {
		return err
	}
	controls, err := engine.LoadFile("qrc:/assets/main.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)
	window.Show()
	window.Wait()
	return nil
}
