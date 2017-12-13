package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/blang/semver"
	"github.com/savaki/jq"
)

const (
	githubV3APIHeader = "application/vnd.github.v3+json"
	latestReleaseURL  = "https://api.github.com/repos/CoveredCI/testman/releases/latest"
	// jqQuery = "{tag:.tag_name, bins:.assets|map({(.name): .browser_download_url})|add}"
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

	if resp.StatusCode == 403 {
		log.Println("[ERR] hit GitHub.com's rate limit, bypassing check")
		latest = binVersion
		return
	}

	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if latest, err = execJQ(body); err != nil {
		return
	}

	if latest[0] == 'v' {
		latest = latest[1:]
	}
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
