package main

import (
	"bytes"
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
	exe := nameExe()
	relURL := releaseDownloadURL + latest + "/" + exe
	sumsURL := releaseDownloadURL + latest + "/sha256s.txt"

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
	fmt.Println(updateID())

	sum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[NFO] checksumed: %s", sum)
	fmt.Println("Fetching checksum...")
	latestSum, err := fetchLatestSum(sumsURL, exe)
	if err != nil {
		return
	}
	if latestSum != sum {
		err = fmt.Errorf("checksums did not match")
		log.Println("[ERR]", err)
		fmt.Println("Data was corrupted!")
		return
	}

	err = os.Rename(updateID(), replacementDst())
	return
}

func nameExe() (exe string) {
	exe = binName + "-" + unameS(runtime.GOOS) + "-" + unameM(runtime.GOARCH)
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return
}

func replacementDst() (binary string) {
	binary, err := exec.LookPath(binName)
	if err != nil {
		binary = os.Args[0]
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

func fetchLatestSum(URL, exe string) (sum string, err error) {
	log.Printf("[NFO] fetching checksum from %s\n", URL)
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

	sums, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	suffix := []byte("  " + exe)
	for _, line := range bytes.Split(sums, []byte{'\n'}) {
		if bytes.HasSuffix(line, suffix) {
			sum = string(bytes.TrimSuffix(line, suffix))
			log.Printf("[NFO] got checksum: %s\n", sum)
			return
		}
	}
	err = fmt.Errorf("%s not found in body", suffix)
	log.Println("[ERR]", err)
	return
}
