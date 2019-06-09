package main_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/sha3"
)

const (
	binPath    = "bin/netflix-passwd-rotate"
	credPath   = "test-data/netflix"
	testPwPath = "nflx-pw-test"
	testKeyEnv = "TEST_KEY"
)

// nflxCreds has the credentials for test logins.
type nflxCreds struct {
	UsernameFix string `json:"username_fix"`
	PasswordOld string `json:"password_old"`
	PasswordNew string `json:"password_new"`
}

// nflxEnc is for JSON encoding the struct.
type nflxEnc struct {
	Netflix nflxCreds `json:"netflix"`
}

// execParams is a struct for running tests.
type execParams struct {
	flags  []string // Flags to be passed to the CLI.
	output string   // Expected output (case insensitive, substring).
	file   string   // Output file to write the password to.
	status int      // Expected exit status code.

	useOld  bool // Use the old password as the new password.
	swapOld bool // Swap the old password with the new password.

	// Use the password from the previous test run
	// (e.g., if the previous run wrote the password to a file).
	prevPword bool

	fileIdx  int // Index of the file path in `flags'.
	unameIdx int // Index of the username in `flags'.
	oldPwIdx int // Index of the old password in `flags'.
	newPwIdx int // Index of the old password in `flags'.

	comment string // What the test does.
}

// Global variables.
var (
	binary      string
	credentials string
	tmpDir      string
	prevPword   string
	login       nflxEnc
)

// All the test cases go here.
var CmdTests = []execParams{
	execParams{
		flags: []string{
			"-usrname", "foo",
			"-old-pword", "bar",
			"-new-passwd", "baz",
			"-no-color",
		},
		output:  "flag provided but not defined",
		status:  2,
		comment: "Test flags/CLI validation.",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "foo",
			"-new-password", "bar",
			"-no-color",
		},
		output:   "ERR: Your password must contain between 4 and 60 characters.",
		status:   2,
		unameIdx: 1,
		comment:  "Test a bad password.",
	},
	execParams{
		flags: []string{
			"-username", "foo",
			"-old-password", "bar",
			"-new-password", "baz",
			"-no-color",
			"-test",
		},
		output:  "ERR: Please enter a valid email.",
		status:  2,
		comment: "Test a bad email address.",
	},
	execParams{
		flags: []string{
			"-username", "42",
			"-old-password", "bar",
			"-new-password", "baz",
			"-no-color",
			"-test",
		},
		output:  "ERR: Please enter a valid phone number.",
		status:  2,
		comment: "Test a bad phone number.",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "bar123",
			"-new-password", "baz123",
			"-no-color",
			"-test",
		},
		output: "ERR: Incorrect password. " +
			"Please try again or you can reset your password.",
		status:   2,
		unameIdx: 1,
		comment:  "Test an invalid password.",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "stub",
			"-new-password", "stub",
			"-no-color",
			"-test",
		},
		output: "ERR: Sorry, you cannot use a previous password. " +
			"Please try another password.",
		status:   2,
		unameIdx: 1,
		oldPwIdx: 3,
		newPwIdx: 5,
		useOld:   true,
		comment:  "Test login success.",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "stub",
			"-new-password", "stub",
			"-no-color",
			"-test",
		},
		output:   "INF: The password for Netflix was updated successfully!",
		status:   0,
		unameIdx: 1,
		oldPwIdx: 3,
		newPwIdx: 5,
		comment:  "Test reset success.",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "stub",
			"-auto-generate",
			"-out-file", "stub",
			"-no-color",
			"-test",
		},
		output:   "INF: The password for Netflix was updated successfully!",
		file:     "stub",
		status:   0,
		unameIdx: 1,
		oldPwIdx: 3,
		fileIdx:  6,
		swapOld:  true,
		comment:  "Test reset success (with auto generation).",
	},
	execParams{
		flags: []string{
			"-username", "stub",
			"-old-password", "stub",
			"-new-password", "stub",
			"-no-color",
			"-test",
		},
		output:    "INF: The password for Netflix was updated successfully!",
		status:    0,
		unameIdx:  1,
		oldPwIdx:  3,
		newPwIdx:  5,
		useOld:    true,
		swapOld:   true,
		prevPword: true,
		comment:   "Test reset success (with password from file).",
	},
}

// getPath gets the paths of files under this directory.
func getPath(bin string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, bin), nil
}

// getKey generates a key for AES-256-GSM encryption/decryption.
func getKey(passwd []byte) [32]byte {
	return sha3.Sum256(passwd)
}

// getNonce generates a nonce for AES-256-GSM encryption/decryption.
func getNonce(passwd []byte) [12]byte {
	var hash = [12]byte{}
	sha3.ShakeSum256(hash[:], passwd)
	return hash
}

// decrypt decrypts the Netflix credentials file (into a struct).
func decrypt(key [32]byte, nonce [12]byte, path string) (nflxEnc, error) {
	var (
		encrypted []byte
		plaintext []byte
		block     cipher.Block
		aesgcm    cipher.AEAD
		data      nflxEnc
		err       error
	)

	if encrypted, err = ioutil.ReadFile(path); err != nil {
		return nflxEnc{}, err
	}

	if block, err = aes.NewCipher(key[:]); err != nil {
		return nflxEnc{}, err
	}

	if aesgcm, err = cipher.NewGCM(block); err != nil {
		return nflxEnc{}, err
	}

	if plaintext, err = aesgcm.Open(nil, nonce[:], encrypted, nil); err != nil {
		return nflxEnc{}, err
	}

	if err = json.Unmarshal(plaintext, &data); err != nil {
		return nflxEnc{}, err
	}

	return data, nil
}

// encrypt encrypts the credentials struct into a file.
// Note:
//      To run tests, use this function to encrypt a JSON file with the
//      Netflix account credentials (a test account) with a password.
//      The `key' and `nonce' can be generated by `getKey', `getNonce'
//      with a password of your choice. Make sure to update `credPath'
//      if you're using a different location to store the encrypted file.
func encrypt(key [32]byte, nonce [12]byte, creds nflxEnc, path string) error {
	var (
		plaintext []byte
		encrypted []byte
		block     cipher.Block
		aesgcm    cipher.AEAD
		file      *os.File
		err       error
	)

	if plaintext, err = json.Marshal(creds); err != nil {
		return err
	}

	if block, err = aes.NewCipher(key[:]); err != nil {
		return err
	}

	if aesgcm, err = cipher.NewGCM(block); err != nil {
		return err
	}

	encrypted = aesgcm.Seal(nil, nonce[:], plaintext, nil)

	if file, err = os.Create(path); err != nil {
		return err
	}

	if _, err = file.Write(encrypted); err != nil {
		return err
	}

	return nil
}

// readPass reads the password from a file.
func readPass(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

// exCmd executes a the binary.
func exCmd(p execParams) (*exec.Cmd, []byte, error) {
	var (
		cmd *exec.Cmd
		err error
		out []byte
	)

	cmd = exec.Command(binary, p.flags...)
	out, err = cmd.CombinedOutput()

	return cmd, out, err
}

// Some things to load before we start the test.
func init() {
	var (
		err   error
		pword string
		key   [32]byte
		nonce [12]byte
	)

	if binary, err = getPath(binPath); err != nil {
		log.Fatalf("path: unable to find the binary: %s\n", err)
	}

	if credentials, err = getPath(credPath); err != nil {
		log.Fatalf("path: unable to find credentials: %s\n", err)
	}

	pword = os.Getenv(testKeyEnv)
	if pword == "" {
		log.Fatalf("env: unable to load test key\n")
	}

	key = getKey([]byte(pword))
	nonce = getNonce([]byte(pword))

	login, err = decrypt(key, nonce, credentials)
	if err != nil {
		log.Fatalf("decrypt: bad credentials file: %s\n", err)
	}
}

// TestMain tests the CLI.
func TestMain(t *testing.T) {
	var (
		test  execParams
		err   error
		exErr *exec.ExitError
		ok    bool
		tmp   []byte
	)

	tmpDir, err = ioutil.TempDir("", testPwPath)
	if err != nil {
		t.Fatalf("error: unable to create a temporary directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, test = range CmdTests {
		// Some values need to be assigned during runtime.
		if test.unameIdx != 0 {
			test.flags[test.unameIdx] = login.Netflix.UsernameFix
		}

		if test.oldPwIdx != 0 {
			if test.prevPword {
				test.flags[test.oldPwIdx] = prevPword
			} else if test.swapOld {
				test.flags[test.oldPwIdx] = login.Netflix.PasswordNew
			} else {
				test.flags[test.oldPwIdx] = login.Netflix.PasswordOld
			}
		}

		if test.newPwIdx != 0 {
			if test.useOld {
				test.flags[test.newPwIdx] = login.Netflix.PasswordOld
			} else {
				test.flags[test.newPwIdx] = login.Netflix.PasswordNew
			}
		}

		if test.fileIdx != 0 {
			test.flags[test.fileIdx] = filepath.Join(tmpDir, "pw-file")
			test.file = filepath.Join(tmpDir, "pw-file")
		}

		_, tmp, err = exCmd(test)
		if err != nil {
			if exErr, ok = err.(*exec.ExitError); ok {
				if exErr.ExitCode() != test.status ||
					!strings.Contains(
						strings.ToLower(string(tmp)),
						strings.ToLower(test.output),
					) {
					t.Fatalf(
						"\nComment: %s\n"+
							"\nstatus:\n\twant:\t%d\n\tgot:\t%d"+
							"\noutput:\n\twant:\t\"%s\"\n\tgot:\t\"%s\"\n",
						test.comment,
						test.status, exErr.ExitCode(),
						test.output, strings.TrimSuffix(
							strings.TrimSpace(string(tmp)), "\n",
						),
					)
				}
			} else {
				t.Fatalf("error: unable to fetch exit status\n")
			}
		} else {
			if test.status != 0 {
				t.Fatalf(
					"expected to fail, but passed: \"%s\"\n",
					strings.TrimSuffix(
						strings.TrimSpace(string(tmp)), "\n",
					),
				)
			}
		}

		if test.file != "" {
			tmp, err = readPass(test.file)
			if err != nil {
				t.Fatalf("error: unable to fetch stored password: %s\n", err)
			}

			prevPword = strings.TrimSuffix(string(tmp), "\n")
		}
	}
}
