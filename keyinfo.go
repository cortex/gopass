package main

import (
	"crypto/rsa"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/proglottis/gpgme"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// GPGME gets very sad when run from lots of goroutines at the same time
// this lock ensures that all operations are serialized.
var gpgmeMutex sync.Mutex

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
func keyInfo(path string) KeyInfo {
	gpgmeMutex.Lock()
	defer gpgmeMutex.Unlock()
	// Find the keyID for the encrypted data
	encKeyID := findKey(path)

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
