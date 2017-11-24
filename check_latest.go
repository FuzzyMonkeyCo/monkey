package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"gopkg.in/blang/semver.v3"
	"gopkg.in/savaki/jq.v0"
)

const (
	githubV3APIHeader = "application/vnd.github.v3+json"
	latestReleaseURL  = "https://api.github.com/repos/CoveredCI/testman/releases/latest"
	// jqQuery = "{tag:.tag_name, bins:.assets|map({(.name): .browser_download_url})|add}"
	jqQuery = ".tag_name"
)

func getLatestRelease() (string, error) {
	get, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		log.Println("[ERR] ", err)
		return "", err
	}

	get.Header.Set("Accept", githubV3APIHeader)
	resp, err := clientUtils.Do(get)
	if err != nil {
		log.Println("[ERR]", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("not 200: %v", resp.Status)
		log.Println("[ERR]", err)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return "", err
	}

	latest, err := execJQ(body)
	if err != nil {
		return "", err
	}

	if latest[0] == 'v' {
		return latest[1:], nil
	}
	return latest, nil
}

func execJQ(body []byte) (string, error) {
	op, err := jq.Parse(jqQuery)
	if err != nil {
		log.Println("[ERR]", err)
		return "", err
	}

	ret, err := op.Apply(body)
	if err != nil {
		log.Println("[ERR]", err)
		return "", err
	}

	res := string(ret)
	return res[1 : len(res)-1], nil
}

func isOutOfDate(current, latest string) (bool, error) {
	vCurrent, err := semver.Make(current)
	if err != nil {
		log.Println("[ERR]", err)
		return false, err
	}

	vLatest, err := semver.Make(latest)
	if err != nil {
		log.Println("[ERR]", err)
		return false, err
	}

	return vLatest.GT(vCurrent), nil
}
