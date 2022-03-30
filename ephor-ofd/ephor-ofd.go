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
	"regexp"
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
		Fields           struct {
			Fp, Fd, Fn string
		}
	}
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

/* Ofd structure, make requests to OFD and build Output */
type Ofd struct {
	Auth struct {
		Token string
	}
	Config struct {
		Host, Login, Password string
	}
	ReceiptId string
	Outcome   Outcome
	Response  Response
}

func (o *Ofd) Init(data Data) {

	counter.Add()
	o.Outcome.Imei = data.Imei
	o.Outcome.MakeEvents(data.Events)
	o.Config = data.ConfigFR
}

func (o *Ofd) MakeAuth() {

	url := fmt.Sprintf("https://%s/api/Authorization/CreateAuthToken", o.Config.Host)
	json_str := fmt.Sprintf(`{"Login":"%s", "Password": "%s"}`, o.Config.Login, o.Config.Password)
	o.Call("POST", url, []byte(json_str))
	if o.Response.Code != 200 {
		o.Outcome.Send()
	}
	o.Auth.Token = o.Response.GetDataString("AuthToken")
	if len(o.Auth.Token) == 0 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = fmt.Sprintf("Error: %s No auth token", cfg.Name)
		o.Outcome.Send()
	}
}

func (o *Ofd) PushData(fiscal_data Fields) {

	json_request, _ := json.Marshal(fiscal_data)
	url := fmt.Sprintf("https://%s/api/kkt/cloud/receipt?AuthToken=%s", o.Config.Host, o.Auth.Token)
	o.Call("POST", url, json_request)

	if o.Response.Code != 200 {
		if o.Response.Error.Code == 1019 {
			re := regexp.MustCompile("--([a-f0-9-]+)--")
			match := re.FindStringSubmatch(o.Response.Error.Message)
			if len(match) > 1 {
				o.ReceiptId = match[1]
				o.GetStatus()
			}
			return
		}
		o.Outcome.Send()
	}
	o.ReceiptId = o.Response.GetDataString("ReceiptId")
	if len(o.ReceiptId) == 0 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = fmt.Sprintf("Error: %s No ReceiptId", cfg.Name)
		o.Outcome.Send()
	}
}

func (o *Ofd) GetStatus() {

	json_str := fmt.Sprintf(`{"Request":{"ReceiptId": "%s"}}`, o.ReceiptId)
	url := fmt.Sprintf("https://%s/api/kkt/cloud/status?AuthToken=%s", o.Config.Host, o.Auth.Token)
	o.Call("POST", url, []byte(json_str))
	status := o.Response.GetDataString("StatusName")

	if status == "CONFIRMED" {

		device := o.Response.GetDataMap("Device")
		o.Outcome.Data.Fields.Fp, _ = device["FPD"].(string)
		o.Outcome.Data.Fields.Fd, _ = device["FDN"].(string)
		o.Outcome.Data.Fields.Fn, _ = device["FN"].(string)

		o.Outcome.Data.Status = "success"
		o.Outcome.Send()

	} else if status == "KKT_ERROR" {
		o.Outcome.Data.Message = o.Response.GetDataString("StatusMessage")
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Send()
	}
}

//TODO: Create interface
func (o *Ofd) Call(method string, url string, json_request []byte) {

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
	json.Unmarshal([]byte(body), &o.Response)

	if cfg.Debug {
		log.Println(resp)
		log.Print(string(body))
	}
	o.Response.Code, o.Outcome.Data.Code = resp.StatusCode, resp.StatusCode

	if resp.StatusCode != 200 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = o.Response.Error.Message
		o.Outcome.Data.StatusCode = o.Response.Error.Code
	}

}

/* Helpers structures */
type Response struct {
	Code   int
	Status string
	Error  struct {
		Code    int
		Message string
	}
	Data map[string]interface{}
}

func (r Response) GetDataString(field string) string {
	str, _ := r.Data[field].(string)
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
		Host, Login, Password string
	}
	InQueue int
	Fields  Fields
}

type Fields struct {
	Request map[string]interface{}
}

var cfg config.Config
var counter count.Counter

func fiscal_process(timeout chan bool, json_data []byte) {

	var data Data
	var ofd Ofd

	json.Unmarshal(json_data, &data)
	ofd.Init(data)
	ofd.MakeAuth()
	ofd.PushData(data.Fields)

	for {

		select {

		case <-time.After(cfg.SleepMilliSec * time.Millisecond):
			ofd.GetStatus()

		case <-timeout:
			ofd.Outcome.Timeout()
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
