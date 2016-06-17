package main

//go:generate genqrc assets/main.qml assets/logo.svg
import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/atotto/clipboard"
	"github.com/proglottis/gpgme"
	"gopkg.in/qml.v1"
)

// A Password entry in Passwords
type Password struct {
	Name string
	Path string
}

// Passwords is the model for the password UI
type Passwords struct {
	passwords     []Password
	hits          []Password
	query         string
	Len           int
	Prefix        string
	Selected      int
	Status        string
	countingDown  bool
	countdownDone chan bool
}

// Quit the application
func (ps *Passwords) Quit() {
	os.Exit(0)
}

// Up moves the selection up
func (ps *Passwords) Up() {
	if ps.Selected > 0 {
		ps.Selected--
		qml.Changed(ps, &ps.Selected)
	}
}

// Down moves the selection down
func (ps *Passwords) Down() {
	if ps.Selected < ps.Len {
		ps.Selected++
		qml.Changed(ps, &ps.Selected)
	}
}

// Password gets the password at a specific index
func (ps *Passwords) Password(index int) Password {
	if index > len(ps.hits) {
		fmt.Println("Bad password fetch", index, len(ps.hits), ps.Len)
		return Password{}
	}
	return ps.hits[index]
}

// ClearClipboard clears the clipboard
func (ps *Passwords) ClearClipboard() {
	if ps.countingDown {
		ps.countdownDone <- true
	}
	ps.countingDown = true
	t := time.NewTicker(1 * time.Second)
	remaining := 45
	for {
		select {
		case <-ps.countdownDone:
			t.Stop()
			ps.countingDown = false
			return
		case <-t.C:
			ps.setStatus(fmt.Sprintf("Will clear in %d seconds", remaining))
			remaining--
			if remaining <= 0 {
				clipboard.WriteAll("")
				ps.setStatus("Clipboard cleared")
				ps.countingDown = false
				t.Stop()
				return
			}
		}
	}
}

// CopyToClipboard copies the selected password to the system clipboard
func (ps *Passwords) CopyToClipboard() {
	if ps.Selected >= len(ps.hits) {
		ps.setStatus("No password selected")
		return
	}
	out, _ := (ps.hits)[ps.Selected].decrypt()
	firstline, _, _ := bufio.NewReader(out).ReadLine()
	if err := clipboard.WriteAll(string(firstline)); err != nil {
		panic(err)
	}
	ps.setStatus("Copied to clipboard")
	go ps.ClearClipboard()
}

// Query updates the hitlist with the given query
func (ps *Passwords) Query(q string) {
	ps.query = q
	ps.updateHits()
	ps.setStatus(fmt.Sprintf("Matched %d items", ps.Len))
}

func (ps *Passwords) add(p Password) {
	ps.passwords = append(ps.passwords, p)
	ps.updateHits()
	ps.setStatus(fmt.Sprintf("Indexed %d entries", ps.Len))
}

func match(query, candidate string) bool {
	lowerQuery := strings.ToLower(query)
	queryParts := strings.Split(lowerQuery, " ")

	lowerCandidate := strings.ToLower(candidate)

	for _, p := range queryParts {
		if !strings.Contains(
			strings.ToLower(lowerCandidate),
			strings.ToLower(p),
		) {
			return false
		}
	}
	return true

}

func (ps *Passwords) updateHits() {
	qml.Lock()
	ps.hits = nil
	for _, p := range ps.passwords {
		if match(ps.query, p.Name) {
			ps.hits = append(ps.hits, p)
		}
	}
	ps.Len = len(ps.hits)
	if ps.Selected > ps.Len {
		ps.Selected = ps.Len
	}
	qml.Changed(ps, &ps.Len)
	qml.Unlock()
}

func (ps *Passwords) setStatus(s string) {
	ps.Status = s
	qml.Changed(ps, &ps.Status)
}

func (ps *Passwords) indexReset() {
	ps.Len = 0
	ps.Selected = 0
	ps.hits = nil
	ps.passwords = nil
}

func (ps *Passwords) indexFile(path string, info os.FileInfo, err error) error {
	if strings.HasSuffix(path, ".gpg") {
		name := strings.TrimSuffix(strings.TrimPrefix(path, passwords.Prefix+"/"), ".gpg")
		ps.add(Password{Name: name, Path: path})
	}
	return nil
}

func (ps *Passwords) indexAll() {
	qml.Lock()
	ps.indexReset()
	filepath.Walk(ps.Prefix, ps.indexFile)
	qml.Unlock()
}

func (p Password) decrypt() (io.Reader, error) {
	file, _ := os.Open(p.Path)
	defer file.Close()
	return gpgme.Decrypt(file)
}

func (ps *Passwords) watch() {
	c := make(chan notify.EventInfo, 1)
	if err := notify.Watch(ps.Prefix+"/...", c, notify.All); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			<-c
			ps.indexAll()
		}
	}()
}

var passwords Passwords

func run() error {
	qml.SetApplicationName("GoPass")
	engine := qml.NewEngine()
	engine.Context().SetVar("passwords", &passwords)
	controls, err := engine.LoadFile("qrc:/assets/main.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)
	window.Show()
	window.Wait()
	return nil
}

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	passwords.Prefix, err = filepath.EvalSymlinks(
		path.Join(usr.HomeDir, ".password-store"),
	)
	if err != nil {
		log.Fatal(err)
	}
	passwords.indexAll()
	passwords.watch()
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
