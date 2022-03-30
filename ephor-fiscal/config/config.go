package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

/* Configuration structure, load from config.json, global */
type Config struct {
	Port, Address, Recipient, Name, LogFile string
	SleepMilliSec, ExecuteMinutes           time.Duration
	Debug                                   bool
}

func (c *Config) Load() {
	file, _ := os.Open("config.json")
	byteValue, _ := ioutil.ReadAll(file)
	defer file.Close()
	json.Unmarshal(byteValue, &c)
}
