package proxy

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

type Config struct {
	TargetHost             string
	CassetteDir            string
	Episodes               []Episode
	Cassette               string   `json:"cassette"`
	RecordNewEpisodes      bool     `json:"record_new_episodes"`
	DenyUnrecordedRequests bool     `json:"deny_unrecorded_requests"`
	RewriteHostHeader      bool     `json:"rewrite_host_header"`
	MatchHeaders           []string `json:"match_headers"`
}

func (c *Config) Load() error {
	c.Episodes = []Episode{}

	cassetteData, err := ioutil.ReadFile(path.Join(c.CassetteDir, c.Cassette+".json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(cassetteData, &c.Episodes)
}

func (c *Config) Save() error {
	jsonData, err := json.MarshalIndent(&c.Episodes, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(c.CassetteDir, 0700)
	return ioutil.WriteFile(path.Join(c.CassetteDir, c.Cassette+".json"), jsonData, 0700)
}
