package main

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha1"
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
	"sync"

	"github.com/proglottis/gpgme"
	"github.com/rjeczalik/notify"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// GPGME gets very sad when run from lots of goroutines at the same time
// this lock ensures that all operations are serialized.
var gpgmeMutex sync.Mutex

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

type KeyInfo struct {
	GPGAgentKeyInfo
	Algorithm   string
	Fingerprint string
	BitLength   uint16
}

// GPGAgentKeyInfo is used for parsing the data from GPGAgent
type GPGAgentKeyInfo struct {
	KeyGrip     string
	Type        string
	Serial      string
	IDStr       string
	Cached      bool
	Protection  string
	Fingerprint string
	TTL         string
	Flags       string
}

func parseKeyinfo(statusLine string) GPGAgentKeyInfo {
	parts := strings.Split(statusLine, " ")
	return GPGAgentKeyInfo{
		KeyGrip:     parts[0],
		Type:        parts[1],
		Serial:      parts[2],
		IDStr:       parts[3],
		Cached:      parts[4] == "1",
		Protection:  parts[5],
		Fingerprint: parts[6],
		TTL:         parts[7],
		Flags:       parts[8],
	}
}

func (p *Password) isCached() bool {
	ki := p.KeyInfo()
	return ki.Cached
}

var algoNames map[packet.PublicKeyAlgorithm]string

func init() {
	algoNames = map[packet.PublicKeyAlgorithm]string{
		packet.PubKeyAlgoRSA:            "RSA",
		packet.PubKeyAlgoRSAEncryptOnly: "RSA Encrypt only",
		packet.PubKeyAlgoRSASignOnly:    "RSA Sign only",
		packet.PubKeyAlgoElGamal:        "ElGamal",
		packet.PubKeyAlgoDSA:            "DSA",
		packet.PubKeyAlgoECDH:           "ECDH",
		packet.PubKeyAlgoECDSA:          "ECDSA",
	}
}

func algoString(a packet.PublicKeyAlgorithm) string {
	return algoNames[a]
}

// KeyInfo gets the KeyInfo for this password
func (p *Password) KeyInfo() KeyInfo {
	gpgmeMutex.Lock()
	defer gpgmeMutex.Unlock()
	// Find the keyID for the encrypted data
	encKeyID := findKey(p.Path)

	// Extract key from gpgme
	c, _ := gpgme.New()
	allKeys, err := gpgme.NewData()
	if err := c.Export(0, allKeys); err != nil {
		fmt.Printf("error reading all keys %s", err)
	}
	allKeys.Seek(0, 0)
	el, err := openpgp.ReadKeyRing(allKeys)
	if err != nil {
		fmt.Println("Failed to open keyring: ", err.Error())
	}
	encKeys := el.KeysById(encKeyID)

	// Get the keyInfo for the file
	var ki KeyInfo
	if len(encKeys) > 0 {
		theKey := encKeys[0].PublicKey
		k := theKey.PublicKey

		switch k := k.(type) {
		case *rsa.PublicKey:
			out := sha1.Sum(append([]byte{0}, k.N.Bytes()...))
			keygrip := fmt.Sprintf("%X", out)
			c.SetProtocol(gpgme.ProtocolAssuan)
			c.AssuanSend("keyinfo "+keygrip, nil, nil,
				func(status string, args string) error {
					ki.GPGAgentKeyInfo = parseKeyinfo(args)
					return nil
				})
			ki.Fingerprint = theKey.KeyIdShortString()
			ki.Algorithm = algoString(theKey.PubKeyAlgo)
			bl, _ := theKey.BitLength()
			ki.BitLength = bl
		default:
			fmt.Println("Unknown crypto")
		}
	}
	return ki
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

func findKey(keypath string) uint64 {
	r, _ := os.Open(keypath)
	packets := packet.NewReader(r)
	for {
		p, err := packets.Next()
		if err != nil {
			return 0
		}
		switch p := p.(type) {
		case *packet.EncryptedKey:
			return p.KeyId
		}
	}
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
	fmt.Printf("added %s\n", p.Name)
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
