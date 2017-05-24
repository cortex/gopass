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

	"sort"

	"github.com/proglottis/gpgme"
	"github.com/rjeczalik/notify"
)

// PasswordStore keeps track of all the passwords
type PasswordStore struct {
	passwords   map[string]string
	Prefix      string
	subscribers []Subscriber
}

// Subscriber is a callback for changes in the PasswordStore
type Subscriber func(status string)

func decrypt(path string) (io.Reader, error) {
	gpgmeMutex.Lock()
	defer gpgmeMutex.Unlock()
	file, _ := os.Open(path)
	defer file.Close()
	return gpgme.Decrypt(file)
}

// Raw returns the password in encrypted form
func Raw(path string) string {
	file, _ := os.Open(path)
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	return base64.StdEncoding.EncodeToString(data)
}

// Metadata of the password
func Metadata(path string) string {
	out, _ := decrypt(path)
	nr := bufio.NewReader(out)
	nr.ReadString('\n')
	metadata, _ := nr.ReadString('\003')
	return metadata
}

func Password(path string) string {
	decrypted, _ := decrypt(path)
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
	ps.passwords = make(map[string]string)
	ps.indexAll()
	ps.watch()
	return ps
}

// Query the PasswordStore
func (ps *PasswordStore) Query(q string) []string {
	var hits []string
	for pwPath, pwName := range ps.passwords {
		if match(q, pwName) {
			hits = append(hits, pwPath)
		}
	}

	sort.Strings(hits)
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
		ps.add(path)
	}
	return nil
}

func (ps *PasswordStore) index(path string) {
	filepath.Walk(path, ps.indexFile)
}

func (ps *PasswordStore) indexAll() {
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
		ps.remove(eventInfo.Path())
		ps.indexAll()
		ps.publishUpdate("Index updated")
	case notify.Write:
		// Path and Name haven ot changed, ignore.
	}
}

func (ps *PasswordStore) add(path string) {
	ps.passwords[path] = generateName(path, ps.Prefix)
}

func (ps *PasswordStore) remove(path string) {
	delete(ps.passwords, path)
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
