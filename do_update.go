package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

const (
	githubV3APIHeader  = "application/vnd.github.v3+json"
	latestReleaseURL   = "https://api.github.com/repos/" + githubSlug + "/releases/latest"
	releaseDownloadURL = "https://github.com/" + githubSlug + "/releases/download/"
)

func peekLatestRelease() (latest string, err error) {
	get, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	get.Header.Set("Accept", githubV3APIHeader)
	log.Printf("[NFO] fetching latest version from %s\n", latestReleaseURL)
	resp, err := clientUtils.Do(get)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
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

func replaceCurrentRelease(latest string) (err error) {
	relURL := releaseDownloadURL + latest + "/" + nameExe()
	sumsURL := relURL + ".sha256.txt"

	bin, err := os.OpenFile(updateID(), os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer bin.Close()
	hash := sha256.New()
	Y := io.MultiWriter(bin, hash)

	log.Printf("[NFO] fetching %s\n", relURL)
	fmt.Println("Fetching", relURL)
	resp, err := clientUtils.Get(relURL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	if _, err = io.Copy(Y, resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[NFO] checksumed: %s", sum)
	log.Printf("[NFO] fetching checksum from %s\n", sumsURL)
	fmt.Println("Fetching checksum...")
	latestSum, err := fetchLatestSum(sumsURL)
	if err != nil {
		return
	}
	if latestSum != sum {
		err = fmt.Errorf("checksums did not match")
		log.Println("[ERR]", err)
		fmt.Println("Data was corrupted!")
		return
	}

	dst, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Println("[NFO] replacing", dst)
	fmt.Println("Replacing", dst)
	err = os.Rename(updateID(), dst)
	return
}

func nameExe() (exe string) {
	exe = binName + "-" + unameS(runtime.GOOS) + "-" + unameM(runtime.GOARCH)
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return
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

func fetchLatestSum(URL string) (sum string, err error) {
	resp, err := clientUtils.Get(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	line, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if len(line) > 64 {
		sum = string(line[:64])
		log.Printf("[NFO] got checksum: %s\n", sum)
		return
	}
	err = fmt.Errorf("no checksum in %s", line)
	log.Println("[ERR]", err)
	return
}
