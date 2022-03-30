package configEphor

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

/* Configuration structure, load from config.json, global */
type Config struct {
    RabbitMq struct {
        Login, Password, Address, Port string
    }
    Db struct {
       Login,Password,Address,DatabaseName string
       Port  uint16
       PgConnectionPool int 
    }
    Services struct {
        EphorPay struct {
            NameQueue string
            Bank struct {
                Address, Port string
                ExecuteMinutes int // this parametr for time run work with bank
                PollingTime int
            }
        }
    }
    LogFile string
    ExecuteMinutes int // this parametr work execute time for one transaction
    Debug bool
}

func (c *Config) Load() {
	file, _ := os.Open("config.json")
	byteValue, _ := ioutil.ReadAll(file)
	defer file.Close()
	json.Unmarshal(byteValue, &c)
}