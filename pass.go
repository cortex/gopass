package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
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

func (p *Password) Password() (string, error) {
	decrypted, err := p.decrypt()
	if err != nil {
		return "", err
	}
	nr := bufio.NewReader(decrypted)
	password, _ := nr.ReadString('\n')
	return password, nil
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

func (ps *PasswordStore) indexFile(path string, info os.FileInfo, err error) error {
	if strings.HasSuffix(path, ".gpg") {
		name := strings.TrimPrefix(path, ps.Prefix)
		name = strings.TrimSuffix(name, ".gpg")
		name = strings.TrimPrefix(name, "/")
		const MaxLen = 40
		if len(name) > MaxLen {
			name = "..." + name[len(name)-MaxLen:]
		}

		ps.add(Password{Name: name, Path: path})
	}
	return nil
}

func (ps *PasswordStore) indexAll() {
	filepath.Walk(ps.Prefix, ps.indexFile)
}

func (ps *PasswordStore) watch() {
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

func (ps *PasswordStore) add(p Password) {
	ps.passwords = append(ps.passwords, p)
	ps.publishUpdate(fmt.Sprintf("Indexed %d entries", len(ps.passwords)))
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
