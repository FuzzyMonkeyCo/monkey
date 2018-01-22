package main

import (
	"log"
	"net/http"
	"encoding/json"

	"github.com/blang/semver"
	"github.com/savaki/jq"
)

const (
	githubV3APIHeader = "application/vnd.github.v3+json"
	latestReleaseURL  = "https://api.github.com/repos/FuzzyMonkeyCo/monkey/releases/latest"
	// "{tag:.tag_name, bins:.assets|map({(.name): .browser_download_url})|add}"
	jqQuery = ".tag_name"
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
	if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
		log.Println("[ERR]", err)
		return
	}

	latest = data.Version
	return
}

func execJQ(body []byte) (res string, err error) {
	op, err := jq.Parse(jqQuery)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	ret, err := op.Apply(body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	res = string(ret)
	res = res[1 : len(res)-1]
	return
}

func isOutOfDate(current, latest string) (ko bool, err error) {
	vCurrent, err := semver.Make(current)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	vLatest, err := semver.Make(latest)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	ko = vLatest.GT(vCurrent)
	return
}
