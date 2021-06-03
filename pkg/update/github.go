// TODO: generate this with godownloader
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type GithubRelease struct {
	Slug   string
	Name   string
	Client *http.Client
}

func NewGithubRelease(slug, exe string) *GithubRelease {
	return &GithubRelease{
		Slug: slug,
		Name: exe,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (rel *GithubRelease) LatestURL() string {
	return "https://api.github.com/repos/" + rel.Slug + "/releases/latest"
}

func (rel *GithubRelease) DownloadURL() string {
	return "https://github.com/" + rel.Slug + "/releases/download/"
}

func (rel *GithubRelease) PeekLatestRelease() (latest string, err error) {
	get, err := http.NewRequest(http.MethodGet, rel.LatestURL(), nil)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	get.Header.Set("Accept", "application/vnd.github.v3+json")
	log.Printf("[NFO] fetching latest version from %s", rel.LatestURL())
	resp, err := rel.Client.Do(get)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = newStatusError(http.StatusOK, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	var data struct {
		Version string `json:"tag_name"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("[ERR]", err)
		return
	}

	latest = data.Version
	return
}

// assumes not v-prefixed
// assumes never re-tagging releases
// assumes only releasing newer tags
func (rel *GithubRelease) ReplaceCurrentRelease(latest string) (err error) {
	relURL := rel.DownloadURL() + latest + "/" + rel.archive()
	sumsURL := rel.DownloadURL() + latest + "/checksums.sha256.txt"

	log.Printf("[NFO] fetching %s", relURL)
	fmt.Println("Fetching", relURL)
	resp, err := rel.Client.Get(relURL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = newStatusError(http.StatusOK, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	var rchiv bytes.Buffer
	hash := sha256.New()
	Y := io.MultiWriter(&rchiv, hash)

	tee := io.TeeReader(resp.Body, Y)

	var zr *gzip.Reader
	if zr, err = gzip.NewReader(tee); err != nil {
		log.Println("[ERR]", err)
		return
	}
	tr := tar.NewReader(zr)

	var dst string
	if dst, err = exec.LookPath(os.Args[0]); err != nil {
		log.Println("[ERR]", err)
		return
	}
	// Create tmp file in hopefully the same filesystem as dst
	distDirname := filepath.Dir(dst)

	exe := rel.Executable()
	var bin *os.File
	if bin, err = ioutil.TempFile(distDirname, exe); err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer os.Remove(bin.Name())
	defer bin.Close()

	for {
		var header *tar.Header
		header, err = tr.Next()
		switch err {
		case nil:
		case io.EOF:
			break
		default:
			log.Println("[ERR]", err)
			return
		}

		if header.Typeflag == tar.TypeReg && header.Name == rel.Name {
			if _, err = io.CopyN(bin, tr, header.Size); err != nil {
				log.Println("[ERR]", err)
				return
			}
			break
		}
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[NFO] checksumed %s", sum)
	log.Printf("[NFO] fetching checksum from %s", sumsURL)
	fmt.Println("Fetching checksum...")
	latestSum, err := rel.fetchLatestSum(sumsURL, exe)
	if err != nil {
		return
	}
	if latestSum != sum {
		err = errors.New("checksums did not match")
		log.Println("[ERR]", err)
		fmt.Println("Data was corrupted!")
		return
	}

	log.Println("[NFO] replacing", dst)
	fmt.Println("Replacing", dst)
	if err = bin.Chmod(os.FileMode(0744)); err != nil {
		log.Println("[ERR]", err)
		return
	}
	if err = os.Rename(bin.Name(), dst); err != nil {
		log.Println("[ERR]", err)
		return
	}
	err = nil
	return
}

func (rel *GithubRelease) Executable() (exe string) {
	exe = rel.Name + "-" + unameS(runtime.GOOS) + "-" + unameM(runtime.GOARCH)
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return
}

func (rel *GithubRelease) archive() string {
	name := rel.Name + "-" + unameS(runtime.GOOS) + "-" + unameM(runtime.GOARCH)
	if runtime.GOOS == "windows" {
		return name + ".zip"
	}
	return name + ".tar.gz"
}

// Sync with https://github.com/mitchellh/gox/pull/103
func unameS(os string) string {
	return map[string]string{
		"darwin":    "Darwin",
		"dragonfly": "DragonFly",
		"freebsd":   "FreeBSD",
		"linux":     "Linux",
		"netbsd":    "NetBSD",
		"openbsd":   "OpenBSD",
		"plan9":     "Plan9",
		"solaris":   "SunOS",
		"windows":   "Windows",
	}[os]
}

// Sync with https://github.com/mitchellh/gox/pull/103
func unameM(arch string) string {
	return map[string]string{
		"386":     "i386",
		"amd64":   "x86_64",
		"arm":     "arm",
		"arm64":   "aarch64",
		"ppc64":   "ppc64",
		"ppc64le": "ppc64le",
	}[arch]
}

func (rel *GithubRelease) fetchLatestSum(URL, exe string) (sum string, err error) {
	resp, err := rel.Client.Get(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = newStatusError(http.StatusOK, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for _, line := range strings.Split(string(contents), "\n") {
		if strings.Contains(line, exe) {
			sum = line[:64]
			log.Printf("[NFO] expecting checksum %q", sum)
			return
		}
	}
	err = fmt.Errorf("no checksum in %q", contents)
	log.Println("[ERR]", err)
	return
}

func newStatusError(expectedCode int, got string) error {
	return fmt.Errorf("expected status %d but got %q", expectedCode, got)
}
