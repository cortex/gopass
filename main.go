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

	"github.com/atotto/clipboard"
	"github.com/proglottis/gpgme"
	"gopkg.in/qml.v1"
)

type Password struct {
	Name string
	Path string
}

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

func (ps *Passwords) Up() {
	if ps.Selected > 0 {
		ps.Selected--
		qml.Changed(ps, &ps.Selected)
	}
}

func (ps *Passwords) Down() {
	if ps.Selected < ps.Len {
		ps.Selected++
		qml.Changed(ps, &ps.Selected)
	}
}

func (ps *Passwords) Add(p Password) {
	ps.passwords = append(ps.passwords, p)
	ps.updateHits()
	ps.SetStatus(fmt.Sprintf("Indexed %d entries", ps.Len))
}

func (ps *Passwords) Password(index int) Password {
	return ps.hits[index]
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
}

func (ps *Passwords) SetStatus(s string) {
	ps.Status = s
	qml.Changed(ps, &ps.Status)
}

func (ps *Passwords) Query(q string) {
	ps.query = q
	ps.updateHits()
	ps.SetStatus(fmt.Sprintf("Matched %d items", ps.Len))
}

func (ps *Passwords) killCountdown() {

}

func (ps *Passwords) ClearClipboard() {
	if ps.countingDown {
		ps.countdownDone <- true
	}
	ps.countingDown = true
	t := time.NewTicker(1 * time.Second)
	remaining := 5
	for {
		select {
		case <-ps.countdownDone:
			t.Stop()
			ps.countingDown = false
			return
		case <-t.C:
			ps.SetStatus(fmt.Sprintf("Will clear in %d seconds", remaining))
			remaining--
			if remaining <= 0 {
				clipboard.WriteAll("")
				ps.SetStatus("Clipboard cleared")
				ps.countingDown = false
				t.Stop()
				return
			}
		}
	}
}

func (ps *Passwords) Copy() {
	if ps.Selected >= len(ps.hits) {
		ps.SetStatus("No password selected")
		return
	}
	out, _ := (ps.hits)[ps.Selected].Decrypt()
	firstline, _, _ := bufio.NewReader(out).ReadLine()
	if err := clipboard.WriteAll(string(firstline)); err != nil {
		panic(err)
	}
	ps.SetStatus("Copied to clipboard")
	go ps.ClearClipboard()
}

func (ps *Passwords) Index(path string, info os.FileInfo, err error) error {
	if strings.HasSuffix(path, ".gpg") {
		name := strings.TrimSuffix(strings.TrimPrefix(path, passwords.Prefix), ".gpg")
		ps.Add(Password{Name: name, Path: path})
	}
	return nil
}

var passwords Passwords

func (p Password) Decrypt() (io.Reader, error) {
	file, _ := os.Open(p.Path)
	defer file.Close()
	return gpgme.Decrypt(file)
}

func run() error {
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
	passwords.Prefix = path.Join(usr.HomeDir, ".password-store")
	filepath.Walk(passwords.Prefix, passwords.Index)
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
