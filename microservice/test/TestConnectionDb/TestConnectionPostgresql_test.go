package TestConnectPostgresql

import (
	"fmt"
	connectionPostgresql "connectionDB/connect"
	"testing"
   "github.com/stretchr/testify/assert"
)

var ConnDb connectionPostgresql.DatabaseInstance
var Config = make(map[string]interface{})

func initConfig() (map[string]interface{}){

   Config["LoginDb"] = "postgres"
   Config["PortDb"] = uint16(5432)
   Config["PasswordDb"] = "123"
   Config["NameDb"] = "postgres"
   Config["Address"] = "127.0.0.1"
   return Config
}

func TestGetConfig(t *testing.T){
   config := initConfig()
   pgConf := ConnDb.GetConfigDb(config["LoginDb"].(string),config["PasswordDb"].(string),config["Address"].(string),config["NameDb"].(string),config["PortDb"].(uint16))
   pgConfType := fmt.Sprintf("%T", pgConf)
   assert.Equal(t,pgConfType,"pgxpool.Config")
}

func TestGetStringConnect(t *testing.T){
  stringPostgres := ConnDb.GetStringConnect(Config["LoginDb"].(string),
                           Config["PasswordDb"].(string),
                           Config["Address"].(string),
                           Config["NameDb"].(string),
                           Config["PortDb"].(uint16))
  testString := "postgres://postgres:123@127.0.0.1:5432/postgres?sslmode=disable&pool_max_conns=10"
  assert.Equal(t,stringPostgres,testString)
}

func TestNewConn(t *testing.T){
   ConnDb.NewConn(10,Config["LoginDb"].(string),
         Config["PasswordDb"].(string),
         Config["Address"].(string),
         Config["NameDb"].(string),
         Config["PortDb"].(uint16))
  testString := "postgres://postgres:123@127.0.0.1:5432/postgres?sslmode=disable&pool_max_conns=10"
  pgConfType := fmt.Sprintf("%T", ConnDb.ConnConfig)
  assert.Equal(t,pgConfType,"pgxpool.Config")
  assert.Equal(t,ConnDb.Url,testString) 
}

func TestReconnectSuccess(t *testing.T){
  conn,_ := ConnDb.Reconnect()
  connType := fmt.Sprintf("%T", conn)
  assert.Equal(t,connType,"*pgxpool.Pool")
}

func TestReconnectFail(t *testing.T){
  ConnDb.Url = "postgres://postgres:0@127.0.0.1:5432/postgres?sslmode=disable&pool_max_conns=10"
  _,err := ConnDb.Reconnect()
  typeError := fmt.Sprintf("%T", err)
  assert.Equal(t,typeError,"*errors.errorString")
}

func TestGetConn(t *testing.T){
  ConnDb.NewConn(10,Config["LoginDb"].(string),
         Config["PasswordDb"].(string),
         Config["Address"].(string),
         Config["NameDb"].(string),
         Config["PortDb"].(uint16))
  pgxPool := ConnDb.GetConn()
  typePgxPool := fmt.Sprintf("%T", pgxPool)
  assert.Equal(t,typePgxPool,"*pgxpool.Pool")
}

func TestBuildString(t *testing.T){
   /*
   test update build string sql
   */
   parametrs := make(map[string]interface{})
   Where := make(map[string]interface{})
   parametrs["id"] = 2
   parametrs["type"] = "67"
   Where["test_1"] = "89"
   paramString,whereString := ConnDb.BuildString(parametrs,Where,"update")
   assert.Equal(t,paramString,` SET "id"=2,"type"=67`)
   assert.Equal(t,whereString,` WHERE "test_1"=89`)
}
