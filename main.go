package main

import (
	"bufio"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/proglottis/gpgme"
	"gopkg.in/qml.v1"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

type Password struct {
	Name string
	Path string
}

type Passwords struct {
	passwords []Password
	hits      []Password
	query     string
	Len       int
	Prefix    string
	Selected  int
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

func (ps *Passwords) Query(q string) {
	ps.query = q
	ps.updateHits()
}

func (ps *Passwords) Copy() {
	fmt.Println("Copied to clipboard")
	out, _ := ps.hits[ps.Selected].Decrypt()
	firstline, _, _ := bufio.NewReader(out).ReadLine()
	if err := clipboard.WriteAll(string(firstline)); err != nil {
		panic(err)
	}
}

var passwords Passwords

func (p Password) Decrypt() (io.Reader, error) {
	file, _ := os.Open(p.Path)
	defer file.Close()
	return gpgme.Decrypt(file)
}

func showPass(path string, info os.FileInfo, err error) error {
	name := strings.TrimSuffix(strings.TrimPrefix(path, passwords.Prefix), ".gpg")
	fmt.Println(name)
	passwords.Add(Password{Name: name, Path: path})
	return nil
}

func run() error {
	engine := qml.NewEngine()
	engine.Context().SetVar("passwords", &passwords)
	controls, err := engine.LoadFile("main.qml")
	if err != nil {
		return err
	}

	window := controls.CreateWindow(nil)

	window.Show()
	window.Wait()
	return nil
}
func main() {
	fmt.Printf("Hello world")
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(usr.HomeDir)
	passwords.Prefix = path.Join(usr.HomeDir, ".password-store")
	filepath.Walk(passwords.Prefix, showPass)
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
