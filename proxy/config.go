package proxy

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

type Config struct {
	CassetteDir string
	Cassette    string `json:"cassette"`
	Episodes    []Episode
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
	jsonData, err := json.Marshal(&c.Episodes)
	if err != nil {
		return err
	}
	os.MkdirAll(c.CassetteDir, 0700)
	return ioutil.WriteFile(path.Join(c.CassetteDir, c.Cassette+".json"), jsonData, 0700)
}
