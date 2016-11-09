package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/proglottis/gpgme"
	"github.com/rjeczalik/notify"
)

// PasswordStore keeps track of all the passwords
type PasswordStore struct {
	passwords   []Password
	Prefix      string
	subscribers []Subscriber
}

// Subscriber is a callback for changes in the PasswordStore
type Subscriber func(status string)

// A Password entry in Passwords
type Password struct {
	Name string
	Path string
}

func (p *Password) decrypt() (io.Reader, error) {
	gpgmeMutex.Lock()
	defer gpgmeMutex.Unlock()
	file, _ := os.Open(p.Path)
	defer file.Close()
	return gpgme.Decrypt(file)
}

// Raw returns the password in encrypted form
func (p *Password) Raw() string {
	file, _ := os.Open(p.Path)
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	return base64.StdEncoding.EncodeToString(data)
}

// Metadata of the password
func (p *Password) Metadata() string {
	out, _ := p.decrypt()
	nr := bufio.NewReader(out)
	nr.ReadString('\n')
	metadata, _ := nr.ReadString('\003')
	return metadata
}

func (p *Password) Password() string {
	decrypted, _ := p.decrypt()
	nr := bufio.NewReader(decrypted)
	password, _ := nr.ReadString('\n')
	return password
}

// NewPasswordStore creates a new password store
func NewPasswordStore() *PasswordStore {
	ps := new(PasswordStore)
	path, err := findPasswordStore()
	if err != nil {
		log.Fatal(err)
	}
	ps.Prefix = path
	ps.indexAll()
	ps.watch()
	return ps
}

// Query the PasswordStore
func (ps *PasswordStore) Query(q string) []Password {
	var hits []Password
	for _, p := range ps.passwords {
		if match(q, p.Name) {
			hits = append(hits, p)
		}
	}
	return hits
}

// Subscribe starts calling cb when anything in the PasswordStore changes
func (ps *PasswordStore) Subscribe(cb Subscriber) {
	ps.subscribers = append(ps.subscribers, cb)
}

func (ps *PasswordStore) publishUpdate(status string) {
	for _, s := range ps.subscribers {
		s(status)
	}
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

func generateName(path, prefix string) string {
	name := strings.TrimPrefix(path, prefix)
	name = strings.TrimSuffix(name, ".gpg")
	name = strings.TrimPrefix(name, "/")
	const MaxLen = 40
	if len(name) > MaxLen {
		name = "..." + name[len(name)-MaxLen:]
	}
	return name
}

func (ps *PasswordStore) indexFile(path string, info os.FileInfo, err error) error {
	if strings.HasSuffix(path, ".gpg") {
		name := generateName(path, ps.Prefix)
		ps.add(Password{Name: name, Path: path})
	}
	return nil
}

func (ps *PasswordStore) index(path string) {
	filepath.Walk(path, ps.indexFile)
}

func (ps *PasswordStore) indexAll() {
	ps.clearAll()
	ps.index(ps.Prefix)
}

func (ps *PasswordStore) watch() {
	c := make(chan notify.EventInfo, 1)
	if err := notify.Watch(ps.Prefix+"/...", c, notify.All); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			ps.updateIndex(<-c)
		}
	}()
}

func (ps *PasswordStore) updateIndex(eventInfo notify.EventInfo) {
	switch eventInfo.Event() {
	case notify.Create:
		ps.index(eventInfo.Path())
		ps.publishUpdate("Entry added")
	case notify.Remove:
		ps.remove(eventInfo.Path())
		ps.publishUpdate("Entry removed")
	case notify.Rename:
		// EventInfo contains old path, but we don't know the new one. Update all
		ps.indexAll()
		ps.publishUpdate("Index updated")
	case notify.Write:
		// Path and Name haven ot changed, ignore.
	}
}

func (ps *PasswordStore) clearAll() {
	ps.passwords = nil
}

func (ps *PasswordStore) add(p Password) {
	ps.passwords = append(ps.passwords, p)
}

func (ps *PasswordStore) remove(path string) {
	for i, p := range ps.passwords {
		if p.Path == path {
			ps.passwords[i] = ps.passwords[len(ps.passwords)-1]
			ps.passwords = ps.passwords[:len(ps.passwords)-1]
			return
		}
	}
}

func findPasswordStore() (string, error) {

	var homeDir string
	if usr, err := user.Current(); err == nil {
		homeDir = usr.HomeDir
	}

	pathCandidates := []string{
		os.Getenv("PASSWORD_STORE_DIR"),
		path.Join(homeDir, ".password-store"),
		path.Join(homeDir, "password-store"),
	}

	for _, p := range pathCandidates {
		var err error
		if p, err = filepath.EvalSymlinks(p); err != nil {
			continue
		}
		if _, err = os.Stat(p); err != nil {
			continue
		}
		return p, nil
	}
	return "", errors.New("Couldn't find a valid password store")
}
