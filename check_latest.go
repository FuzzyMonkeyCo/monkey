package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"gopkg.in/blang/semver.v3"
	"gopkg.in/savaki/jq.v0"
)

const (
	githubV3APIHeader = "application/vnd.github.v3+json"
	latestReleaseURL  = "https://api.github.com/repos/CoveredCI/manlion/releases/latest"
	// jqQuery = "{tag:.tag_name, bins:.assets|map({(.name): .browser_download_url})|add}"
	jqQuery = ".tag_name"
)

func getLatestRelease() string {
	get, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	get.Header.Set("Accept", githubV3APIHeader)
	client := &http.Client{}
	resp, err := client.Do(get)
	if err != nil {
		log.Fatal("!GET: ", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("!200: ", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}

	latest := execJQ(body)
	if latest[0] == 'v' {
		return latest[1:]
	}
	return latest
}

func execJQ(body []byte) string {
	op, err := jq.Parse(jqQuery)
	if err != nil {
		log.Fatal("!jq: ", err)
	}

	ret, err := op.Apply(body)
	if err != nil {
		log.Fatal("!exec jq: ", err)
	}

	res := string(ret)
	return res[1 : len(res)-1]
}

func isOutOfDate(current, latest string) bool {
	vCurrent, err := semver.Make(current)
	if err != nil {
		log.Fatal("!vCurrent: ", err)
	}

	vLatest, err := semver.Make(latest)
	if err != nil {
		log.Fatal("!vLatest: ", err)
	}

	return vLatest.GT(vCurrent)
}
