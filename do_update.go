package main

import (
	"log"
	"net/http"
	"encoding/json"
)

const (
	githubV3APIHeader = "application/vnd.github.v3+json"
	latestReleaseURL  = "https://api.github.com/repos/"+githubSlug+"/releases/latest"
	releaseDownloadURL = "https://github.com/"+githubSlug+"/releases/download/"
)

func getLatestRelease() (latest string, err error) {
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
	// sumsURL := releaseDownloadURL + latest +"/sha256s.txt"
	// sums := 
	return
}
