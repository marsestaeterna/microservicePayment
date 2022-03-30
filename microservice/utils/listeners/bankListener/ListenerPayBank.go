package bankListener

import(
	"encoding/json"
    "fmt"
    "net/http"
    "io/ioutil"
	"time"
    "log"
    configEphor "configEphor"
    factoryBank "factory/bank"
    interfaseBank "interface/bankinterface"
    connectionPostgresql "connectionDB/connect"
)

const (
    SBER = 1
)

const Type_Pay_Prepayment int = 1
const Type_Pay_Maxpayment int = 2 

func InitBank (typeBank int) interfaseBank.Bank {
    switch typeBank {
        case SBER:
        bank := factoryBank.GetBank("Sber")
        return bank
    }
    return nil   
}

func StartСommunicationBank(bankChannel chan bool, json_data []byte){
    var request interfaseBank.Request
    json.Unmarshal(json_data, &request)
    connectDb.AddLog(fmt.Sprintf("%+v",request),"PaymentSystem" ,fmt.Sprintf("%+v",request),"EphorErp")
    bank := InitBank(request.Config.BankType)
    if bank == nil {
        <-bankChannel
    }
    bank.InitBankData(&request,connectDb)
    bank.CreateOrder()
    for {
		select {
		case <-time.After(time.Duration(conf.Services.EphorPay.Bank.PollingTime) * time.Millisecond):
            bank.GetStatusOrder()
		case <-bankChannel:
               bank.Timeout()
			return
		}
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
       
        switch req.Method {

        case "POST":
            json_data, _ := ioutil.ReadAll(req.Body)
            defer req.Body.Close()
            bankChannel := make(chan bool)
            go StartСommunicationBank(bankChannel, json_data)
            go func() {
                select {
                case <-time.After(time.Duration(conf.Services.EphorPay.Bank.ExecuteMinutes) * time.Minute):
                    bankChannel <- true
                }
            }()
            return 
        case "GET":
            fmt.Fprintf(w, "%s: Running\n", )
            log.Println("Running")
        default:
            fmt.Fprintf(w, "Sorry, only POST and GET method is supported.")
        }

}
var connectDb *connectionPostgresql.DatabaseInstance
var conf *configEphor.Config

func StartBank(cfg *configEphor.Config,connectPg *connectionPostgresql.DatabaseInstance){
    connectDb = connectPg
    conf = cfg
    http.HandleFunc("/", handler)
    log.Println("Start Bank..")
    point := fmt.Sprintf("%s:%s",cfg.Services.EphorPay.Bank.Address,cfg.Services.EphorPay.Bank.Port)
    log.Println(point)
	if err := http.ListenAndServe(point, nil); err != nil {
		log.Println(err)
	}
    log.Println(http.ListenAndServe(point, nil))
}



