package connectionPostgresql

import (
	"context"
	"fmt"
	"log"
	"time"
	"strconv"
	"strings"
	pgxpool "github.com/jackc/pgx/v4/pgxpool"
)

type DatabaseInstance struct {
	Conn               *pgxpool.Pool
	ConnConfig         *pgxpool.Config
	MaxConnectAttempts int
	Url string 
	idLog int
}

func (db *DatabaseInstance) GetConfigDb(login,password,address,database string,port uint16) (*pgxpool.Config){
	var pgxConf *pgxpool.Config
	return pgxConf
}

func (db *DatabaseInstance) GetStringConnect(login,password,address,database string,port uint16) string {
	PortInt := int(port)
	PortString := strconv.Itoa(PortInt)
	stringConnection := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=10",login,password,address,PortString,database)
	return stringConnection
}

func (db *DatabaseInstance) NewConn(maxAttempts int,login,password,address,database string,port uint16) {
	uri := db.GetStringConnect(login,password,address,database,port);
	connConfig := db.GetConfigDb(login,password,address,database,port)
	if connConfig != nil {
		db.ConnConfig  = connConfig
	} 
	db.Url = uri
	db.MaxConnectAttempts = maxAttempts
}

func (db *DatabaseInstance) GetConn() ( *pgxpool.Pool, error) {
	var err error

	if db.Conn == nil {
		if db.Conn, err = db.Reconnect(); err != nil {
			log.Fatalf("%s", err)
			return nil,err
		}
	}

	if err = db.Conn.Ping(context.Background()); err != nil {
		attempt := 0
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if attempt >= db.MaxConnectAttempts {
				log.Fatalf("connection failed after %d attempt\n", attempt)
			}
			attempt++

			log.Println("reconnecting...")

			db.Conn, err = db.Reconnect()
			if err == nil {
				return db.Conn,nil
			}

			log.Printf("connection was lost. Error: %s. Waiting for 5 sec...\n", err)
		}
	}

	return db.Conn,nil
}

func (db *DatabaseInstance) Reconnect() (*pgxpool.Pool, error) {
	Conn, err := pgxpool.Connect(context.Background(),db.Url)
	if err != nil {
		return nil, fmt.Errorf("unable to connection to database: %v", err)
	}

	if err = Conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("couldn't ping postgre database: %v", err)
	}

	return Conn, err
}

func failOnError(err error, msg string) {
	if err != nil {
	  log.Fatalf("%s: %s", msg, err)
	}
}

func (db *DatabaseInstance) CloseConnectionDb(){
	db.Conn.Close()
}

func (db *DatabaseInstance) PrepareValue(value,Field1,Field2 interface{}) string {
	result := ""
	switch value.(type) {
		case int:
		result = fmt.Sprintf("%q=%v",Field1,Field2)
		case string:
		result = fmt.Sprintf("%q='%v'",Field1,Field2)
		case nil:
		result = fmt.Sprintf("%q='%v'",Field1,"null")
	}
	return result
}

func (db *DatabaseInstance) BuildString(parametrs map[string]interface{},Where map[string]interface{},typeQuery string) (string,string) {
	WhereString := ""
	StringDataSet := ""
	if len(parametrs) == 0 { return "",""}
	if len(Where) !=0 {
		for key, _ := range Where {
			if WhereString == ""{
				WhereString += fmt.Sprintf(" WHERE %q=%v", key,Where[key])
			} else {
				WhereString += fmt.Sprintf(" AND %q=%v",key,Where[key])
			}
		}
	}
	switch typeQuery {

    case "update":
        for key, _ := range parametrs {
			prepare := db.PrepareValue(parametrs[key],key,parametrs[key])
			if StringDataSet == ""{
				StringDataSet += fmt.Sprintf(" SET %s",prepare)
			} else {
				StringDataSet += fmt.Sprintf(",%s",prepare)
			}
		}
		return StringDataSet,WhereString
    case "insert":
		for key, _ := range parametrs {
			if StringDataSet == ""{
				StringDataSet += fmt.Sprintf("(%v",key)
			} else {
				StringDataSet += fmt.Sprintf(",%v", key)
			}
		}
		StringDataSet += ")"
		for key, _ := range parametrs {
			if StringDataSet == ""{
				StringDataSet += fmt.Sprintf(" VALUES (%v",parametrs[key])
			} else {
				StringDataSet += fmt.Sprintf(",%v",parametrs[key])
			}
		}
		StringDataSet += ")"

		return StringDataSet,WhereString
    case "delete":
		return "",WhereString
    default:
        for key, _ := range parametrs {
			if StringDataSet == ""{
				StringDataSet += fmt.Sprintf("%v",key)
			} else {
				StringDataSet += fmt.Sprintf(",%v", key)
			}
		}
		return StringDataSet,WhereString
    }
	
}

func (db *DatabaseInstance) AddLog(data string, url string,response string,imei string) {
	var runtime int
	var id int
	runtime = 0  
	dt := time.Now()
    date := dt.Format("2006-01-02 15:04:05")
	nsec := dt.UnixNano()
	response = strings.Replace(response, "'", "/", -1)
	stringQuery := fmt.Sprintf("INSERT INTO main.log (address,login,date,request_uri,request_id,request_data,response,runtime,runtime_details) VALUES ('%s','%s','%v','%v',%v,'%s','%v',%v,'%v') RETURNING id;","modem",imei,date,url,nsec,data,response,runtime,' ')
	err := db.Conn.QueryRow(context.Background(),stringQuery).Scan(&id)
	fmt.Println(id)
	if err !=nil {
		log.Println(err)
	}
	db.idLog = id
}

func (db *DatabaseInstance) SetLog(data string) {
	log.Println(data)
	stringConnection := fmt.Sprintf("UPDATE main.log SET response='%v' WHERE id=%v",data,db.idLog)
	_,err := db.Conn.Exec(context.Background(),stringConnection)
	if err  != nil{
		log.Println(err)
	}
}

func (db *DatabaseInstance) Set( table string, parametrs map[string]interface{}, where map[string]interface{}) {
	StringDataSet,WhereString :=db.BuildString(parametrs,where,"update")
	log.Printf(" [data] %s", StringDataSet)
	log.Printf(" [where] %s", WhereString)
	stringConnection := fmt.Sprintf("UPDATE main.%s%s%s",table,StringDataSet,WhereString)
	log.Printf(" [x] %s", stringConnection)
	_,err := db.Conn.Exec(context.Background(),stringConnection)
	if err  != nil{
		errData,_ := fmt.Println(err)
		log.Println(errData)
		db.SetLog(fmt.Sprintf("%s",errData))
	}else{
		db.SetLog("Data processed")
	}
}


func checkModelType(){	
}

func String(str interface{}) string {
	return fmt.Sprintf("%v", str)
}
