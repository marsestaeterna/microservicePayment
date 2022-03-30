package main

import (
	"bytes"
	"encoding/json"
	config "ephor-fiscal/config"
	count "ephor-fiscal/counter"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//"regexp"
	"runtime"
	"strconv"
	"time"
)

/* Result structure, build in process and send to final point */
type Outcome struct {
	Imei string
	Data struct {
		Message, Status  string
		Events           []Event
		Code, StatusCode int
		ReceiptId string
		Authorization string
		IdShift string
		Fields           struct {
			Fp, Fd, Fn string
		}
	}
}

type Cashier struct {
	Data  map[string]interface{}
}

func (c Cashier) GetDataCashierString(field string) string {
	str, _ := c.Data[field].(string)
	return str
}

type Event struct {
	Id string
}

func (out *Outcome) MakeEvents(ev []string) {

	for _, s := range ev {
		out.Data.Events = append(out.Data.Events, Event{Id: s})
	}

}


func (out *Outcome) Finish() {
	counter.Sub()
	runtime.Goexit()
}

func (out Outcome) Send() {

	defer out.Finish()
	json_request, _ := json.Marshal(out.Data)

	dc := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	url := fmt.Sprintf("%s&login=%s&password=12345678&_dc=%s", cfg.Recipient, out.Imei, dc)
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if cfg.Debug {
		log.Println(out.Data)
		log.Println(resp)
		log.Println(string(body))
	}
	return
}

func (out Outcome) Timeout() {
	out.Data.Status = "unsuccess"
	out.Data.Code = 0
	out.Data.Message = fmt.Sprintf("Cancelled by a Timeout of %s", cfg.Name)
	out.Send()
}

/* checkBox structure, make requests to OFD and build Output */
type checkBox struct {
	
	Config struct {
		Host, Auth, RecieptId, Login, Password, KeyLicense, IdShift string
	}
	ReceiptId string
	Outcome   Outcome
	Response  Response
	Cashier    Cashier
}

func (o *checkBox) Auth(){
	url := fmt.Sprintf("https://%s/api/v1/cashier/signin", o.Config.Host)
	json_str := fmt.Sprintf(`{"login":"%s", "password": "%s"}`, o.Config.Login, o.Config.Password)
	o.CallAutch("POST", url, []byte(json_str))

	if o.Response.Code != 200 {
		o.Outcome.Send()
	}
	o.Config.Auth = o.Response.GetDataString("access_token")
	o.Outcome.Data.Authorization = o.Response.GetDataString("access_token")
	
	if len(o.Config.Auth) == 0 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = fmt.Sprintf("Error: %s No auth token", cfg.Name)
		o.Outcome.Send()
	}

}

func (o *checkBox) getShift(){
	url := fmt.Sprintf("https://%s/api/v1/shifts", o.Config.Host)
	json_str := []byte("")
	o.CallStartShift("POST", url, []byte(json_str))
	if o.Response.Code != 202 {
		message := o.Response.GetDataString("message");
		if message == "Касир вже працює з даною касою"{
			return
		}
		o.Outcome.Send()
	}
	o.Outcome.Data.IdShift = o.Response.GetDataString("id")
	
	o.Config.IdShift = o.Response.GetDataString("id")
	
}

func (o *checkBox) checkShift(){
	 
	url := fmt.Sprintf("https://%s/api/v1/cashier/shift", o.Config.Host)
	log.Println(url)
	
	json_str := []byte("")
	o.Call("GET", url, []byte(json_str))
	message := o.Response.GetDataString("message");
	if o.Response.Code != 200{
		
		if message == "Not Found"{
			if len(o.Config.Auth) == 0{
				o.Auth()
			}
			o.getShift()
			o.checkShift()
		}
		if message == "Зміну не відкрито"{
			o.Auth()
			o.getShift()
			o.checkShift()
		}
		if message != "Not Found" ||  message != "Зміну не відкрито"{
			o.Outcome.Send()
		}
	}
	if o.Response.Data == nil {
		if len(o.Config.Auth) == 0{
			o.Auth()
			o.getShift()
			o.checkShift()
		}
		
	}
}

func (o *checkBox) Init(data Data) {

	counter.Add()
	o.Outcome.Imei = data.Imei
	o.Outcome.MakeEvents(data.Events)
	o.Config = data.ConfigFR
}

func (o *checkBox) PushData(fiscal_data []byte) {
	
	if o.Config.RecieptId != "" {
		o.ReceiptId = o.Config.RecieptId
		o.GetStatus()
	} else{
		
		url := fmt.Sprintf("https://%s/api/v1/receipts/sell", o.Config.Host)
		o.Call("POST", url, fiscal_data)
		log.Println(url)
		if o.Response.Code != 201 {
			o.Outcome.Data.Message = o.Response.GetDataString("message")
			log.Println(o.Outcome.Data.Message)
			if o.Outcome.Data.Message == "Зміну не відкрито"{
				o.getShift()
				o.checkShift()
				o.PushData(fiscal_data);
			}else{
				o.Outcome.Send()
			}
		}
		
		o.ReceiptId = o.Response.GetDataString("id")
		if len(o.ReceiptId) == 0 {
			o.Outcome.Data.Status = "unsuccess"
			o.Outcome.Data.Message = fmt.Sprintf("Error: %s No ReceiptId", cfg.Name)
			o.Outcome.Send()
		}
	}
	
}

func (o *checkBox) GetStatus() {

	json_str := []byte("")
	url := fmt.Sprintf("https://%s/api/v1/receipts/%s", o.Config.Host,o.ReceiptId)
	o.Call("GET", url, []byte(json_str))
	status := o.Response.GetDataString("status")
	if o.Response.Code == 200 {

		if status == "DONE" {

			serial := o.Response.GetDataIntToString("serial")
			FC := o.Response.GetDataString("fiscal_code")
			FN := o.Response.GetDataMap("shift")
			Casher := FN["cash_register"].(map[string]interface{})
			o.Cashier.Data = Casher
			Number := o.Cashier.GetDataCashierString("fiscal_number")
			
			o.Outcome.Data.Fields.Fp = FC
			o.Outcome.Data.Fields.Fd = serial
			o.Outcome.Data.Fields.Fn = Number
	
			o.Outcome.Data.Status = "success"
			o.Outcome.Data.ReceiptId = o.ReceiptId
			o.Outcome.Send()
	
		} else if status == "ERROR" {
			o.Outcome.Data.Message = o.Response.GetDataString("message")
			o.Outcome.Data.Status = "unsuccess"
			o.Outcome.Send()
		}

	} else{
		o.Outcome.Data.Message = o.Response.GetDataString("message")
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Send()
	}
}

func (o *checkBox) CallAutch(method string, url string, json_request []byte) {
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		o.Outcome.Data.Code = 0
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Send()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal([]byte(body), &o.Response.Data)
	if cfg.Debug {
		log.Println(resp)
		log.Print(string(body))
	}
	o.Response.Code, o.Outcome.Data.Code = resp.StatusCode, resp.StatusCode

	if resp.StatusCode != 201 || resp.StatusCode !=200 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = o.Response.Message
		o.Outcome.Data.StatusCode = 4
	}

}

func (o *checkBox) CallStartShift(method string, url string, json_request []byte) {
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.Config.Auth))
	req.Header.Set("X-License-Key", fmt.Sprintf("%s", o.Config.KeyLicense))
	log.Println(req.Header)
	req.Close = true

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		o.Outcome.Data.Code = 0
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Send()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal([]byte(body), &o.Response.Data)
	if cfg.Debug {
		log.Println(resp)
		log.Print(string(body))
	}
	o.Response.Code, o.Outcome.Data.Code = resp.StatusCode, resp.StatusCode

	if resp.StatusCode != 202 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = o.Response.Message
		o.Outcome.Data.StatusCode = 4
	}
}

//TODO: Create interface
func (o *checkBox) Call(method string, url string, json_request []byte) {
	
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.Config.Auth))
	req.Close = true
	log.Println(req.Header)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		o.Outcome.Data.Code = 0
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Send()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal([]byte(body), &o.Response.Data)
	if cfg.Debug {
		log.Println(resp)
		log.Print(string(body))
	}
	o.Response.Code, o.Outcome.Data.Code = resp.StatusCode, resp.StatusCode
	log.Println(resp)
	if resp.StatusCode != 201 || resp.StatusCode !=200 || resp.StatusCode !=202 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = o.Response.Message
		o.Outcome.Data.StatusCode = 4
	}

}

/* Helpers structures */
type Response struct {
	Code   int
	Message string 
	Data   map[string]interface{}
}

func (r Response) GetDataString(field string) string {
	str, _ := r.Data[field].(string)
	return str
}

func (r Response) GetDataIntToString(field string) string {
	str := fmt.Sprint(r.Data[field])
	return str
}

func (r Response) GetDataMap(field string) map[string]interface{} {
	i, _ := r.Data[field].(map[string]interface{})
	return i
}

type Data struct {
	Events   []string
	Imei     string
	ConfigFR struct {
		Host, Auth, RecieptId, Login, Password, KeyLicense, IdShift string
	}
	InQueue int
	Fields  struct {
		Request json.RawMessage
	}
}

var cfg config.Config
var counter count.Counter

func fiscal_process(timeout chan bool, json_data []byte) {

	var data Data
	var checkbox checkBox

	json.Unmarshal(json_data, &data)
	checkbox.Init(data)
	if len(checkbox.Config.Auth) == 0 {
		checkbox.Auth()
		checkbox.getShift()
	} else{
		checkbox.Outcome.Data.Authorization = checkbox.Config.Auth
		checkbox.checkShift()
	}
	checkbox.PushData(data.Fields.Request)

	for {

		select {

		case <-time.After(cfg.SleepMilliSec * time.Millisecond):
			checkbox.GetStatus()

		case <-timeout:
			checkbox.Outcome.Timeout()
			return
		}
	}
}

func handler(w http.ResponseWriter, req *http.Request) {

	switch req.Method {

	case "POST":
		json_data, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		timeout := make(chan bool)

		go fiscal_process(timeout, json_data)
		go func() {
			select {
			case <-time.After(cfg.ExecuteMinutes * time.Minute):
				timeout <- true
			}
		}()
		return

	case "GET":
		fmt.Fprintf(w, "%s: Running %d\n", cfg.Name, counter.N)

	default:
		fmt.Fprintf(w, "Sorry, only POST and GET method is supported.")
	}
}

func main() {

	cfg.Load()

	http.HandleFunc("/", handler)

	fmt.Println("Starting service...")

	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Panic(err)
		}
		log.SetOutput(file)

	}
	log.Printf("%s Configuration load\n", cfg.Name)
	point := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)
	if err := http.ListenAndServe(point, nil); err != nil {
		log.Panic(err)
	}

}
