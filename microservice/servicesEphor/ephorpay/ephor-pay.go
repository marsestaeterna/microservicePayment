package ephorpay

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	//"reflect"
	connectionPostgresql "connectionDB/connect"
	counter "connectionDB/counter"
	configEphor "configEphor"
	ConnectionRabbitMQ "lib-rabbitmq"
	listenBank "listeners/bankListener"
)
	// -- Vend err -- //
	const VendError_VendFailed 			= 768; // Ошибка выдачи продукта /768
	const VendError_SessionCancelled 	= 769; // 769 оплата отменена автоматом
	const VendError_SessionTimeout 		= 770; // 770 оплата отменена автоматом
	const VendError_WrongProduct 		= 771; // 771 выбрали не тот продукт
	const VendError_VendCancelled 		= 772; //  выдача отменена автоматом


type Request struct {
	Tid     int
	St 		int
	D       string
	Err 	int
}

func initPayCron(forever chan bool) {
	req := Request{}
	stringQueue := cfg.Services.EphorPay.NameQueue
	msg, _ := ConnectionRabbit.RabbitMQConsume(stringQueue)
	counterGo.Add()
	go func() {
		for d := range msg {
			log.Printf("\n [x] %s", d.Body)
			dataLog := fmt.Sprintf("%s", d.Body)
			err2 := json.Unmarshal(d.Body, &req)

			if err2 != nil {
				errData, _ := fmt.Println(err2)
				log.Println(errData)
				ConnDb.AddLog(dataLog, "EphorPay", fmt.Sprintf("%s", err2),"ephorPayError")
				continue
			}
			ConnDb.AddLog(dataLog,"EphorPay", " ",req.D)
			checkStatusTransaction(&req)
		}
	}()
	counterGo.Sub()
	<-forever
}

var cfg configEphor.Config
var ConnectionRabbit ConnectionRabbitMQ.ChannelMQ
var ConnDb connectionPostgresql.DatabaseInstance
var counterGo counter.Counter

func Start(conf *configEphor.Config, Rabbit *ConnectionRabbitMQ.ChannelMQ, Db *connectionPostgresql.DatabaseInstance) {
	fmt.Println("Start EphorPay...")
	ConnectionRabbit = *Rabbit
	ConnDb = *Db
	cfg = *conf
	forever := make(chan bool)
	go listenBank.StartBank(conf,Db)
	go ListenerSignalOS(forever)
	start(forever)
}

func ListenerSignalOS(forever chan bool) {
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)

	go func() {
		for {
			s := <-signalChanel
			switch s {
			case syscall.SIGTERM:
				fmt.Println("Stop service .")
				log.Println("Stop service .")
				stop(forever)
				forever <- true
				// kill -SIGQUIT XXXX [XXXX - идентификатор процесса для программы]
			case syscall.SIGQUIT:
				fmt.Println("Stop service .")
				log.Println("Stop service .")
				stop(forever)
				forever <- true
			case syscall.SIGKILL:
				log.Println("Stop service .")
				fmt.Println("Stop service .")
				stop(forever)
				forever <- true
			}
		}
	}()
	<-forever
	os.Exit(3)
}

func start(forever chan bool) {
	initPayCron(forever)
}

func stop(forever chan bool) {
	ConnectionRabbit.CloseConnectRabbit()
	if counterGo.N == 0 {
		ConnDb.CloseConnectionDb()
	} else {
		go func() {
			select {
			case <-time.After(10 * time.Second):
				forever <- true
			}
		}()
		ConnDb.CloseConnectionDb()
	}
	os.Exit(3)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}


func getDescriptionErr(err interface {}) string{
	errCode := fmt.Sprintf("%v",err)
	var stringErr string
	 switch errCode {
		 case "768":
		 stringErr = `Ошибка выдачи продукта`
		 fallthrough
		 case "769":
		 stringErr = `Оплата отменена автоматом`
		 fallthrough
		 case "770":
		 stringErr = `Оплата отменена автоматом`
		 fallthrough
		 case "771":
		 stringErr = `Выбрали не тот продукт`
		 fallthrough
		 case "772":
		 stringErr = `Выдача отменена автоматом`
	 }
	 return stringErr
}

func getTransactionMap(req *Request) (map[string]interface{}, map[string]interface{}) {
	Where := make(map[string]interface{})
	parametrs := make(map[string]interface{})
	Where["id"] = req.Tid
	parametrs["status"] = req.St
	parametrs["error"] = req.Err
	return Where,parametrs
}

func checkStatusTransaction(req *Request){
	Where,parametrs := getTransactionMap(req)
	errorCode := parametrs["error"]
	stringErr := getDescriptionErr(errorCode)
	parametrs["error"] = stringErr
	ConnDb.Set("transaction", parametrs, Where)
}
