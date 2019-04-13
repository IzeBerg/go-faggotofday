package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Language struct {
	Registered string `json:"registered"`
	Unregistered string `json:"unregistered"`

	RunFaggot string `json:"run_faggot"`
	MyStatsFaggot string `json:"mystats_faggot"`
	StatsFaggot string `json:"stats_faggot"`

	RunNice string `json:"run_nice"`
	MyStatsNice string `json:"mystats_nice"`
	StatsNice string `json:"stats_nice"`

	ErrorOccurred string `json:"error_occurred"`
}

var Languages = map[string]Language{}

func init() {
	err := filepath.Walk(`lang`, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if data, err := ioutil.ReadFile(path); err == nil {
				lang := Language{}
				if err := json.Unmarshal(data, &lang); err != nil {
					return err
				}
				Languages[strings.Replace(strings.ToLower(info.Name()), `.json`, ``, -1)] = lang
			} else {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}
}
