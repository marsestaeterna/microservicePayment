package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	config "ephor-fiscal/config"
	count "ephor-fiscal/counter"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

/* Result structure, build in process and send to final point */
type Outcome struct {
	Imei string
	Data struct {
		Message, Status,Method  string
		Events           []Event
		Code, StatusCode,Fiscalization int
		Fields           struct {
			Fp, Fn string
			Fd     float64
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

type Orange struct {
	Config struct {
		Host, Cert, Key, Sign, Port string
		Fiscalization int
	}
	CheckId, Inn string
	Outcome      Outcome
	Response     Response
}

func (o *Orange) Init(data Data) {

	counter.Add()
	o.Outcome.Imei = data.Imei
	o.Outcome.Data.Fiscalization = data.ConfigFR.Fiscalization
	o.CheckId = data.CheckId
	o.Inn = data.Inn
	o.Outcome.MakeEvents(data.Events)
	o.Config = data.ConfigFR
}

func (o *Orange) PushData(fiscal_data []byte) {
	o.Outcome.Data.Method = "Check"
	url := fmt.Sprintf("https://%s:%s/api/v2/documents/", o.Config.Host, o.Config.Port)
	o.Call("POST", url, fiscal_data)

	if o.Response.Code == 409 {
		o.GetStatus()
		return
	}

	if o.Response.Code != 201 {
		o.Outcome.Send()
	}

}

func (o *Orange) GetStatus() {
	o.Outcome.Data.Method = "CheckStatus"
	url := fmt.Sprintf("https://%s:%s/api/v2/documents/%s/status/%s", o.Config.Host, o.Config.Port, o.Inn, o.CheckId)

	o.Call("GET", url, []byte(""))

	if o.Response.Code > 299 {
		o.Outcome.Send()
	}

	if o.Response.Code == 200 {
		o.Outcome.Data.Status = "success"
		o.Outcome.Data.Fields.Fp, _ = o.Response.Data["fp"].(string)
		o.Outcome.Data.Fields.Fd, _ = o.Response.Data["documentNumber"].(float64)
		o.Outcome.Data.Fields.Fn, _ = o.Response.Data["fsNumber"].(string)
		o.Outcome.Send()

	}
	return

}

//TODO: Create interface
func (o *Orange) Call(method string, url string, json_request []byte) {

	req, _ := http.NewRequest(method, url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-Signature", o.Config.Sign)
	req.Close = true

	cert, err := tls.X509KeyPair([]byte(o.Config.Cert), []byte(o.Config.Key))
	if err != nil {
		log.Println(err)
	}

	client := &http.Client{}
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

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

	if resp.StatusCode > 299 {
		o.Outcome.Data.Status = "unsuccess"
		o.Outcome.Data.Message = strings.Join(o.Response.Errors[:], "\n")
	}
	return

}

/* Helpers structures */
type Response struct {
	Code   int
	Status string
	Errors []string
	Data   map[string]interface{}
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
	Events             []string
	Imei, CheckId, Inn string
	ConfigFR           struct {
		Host, Cert, Key, Sign, Port string
		Fiscalization int
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
	var orange Orange

	json.Unmarshal(json_data, &data)
	orange.Init(data)
	orange.PushData(data.Fields.Request)

	for {

		select {

		case <-time.After(cfg.SleepMilliSec * time.Millisecond):
			orange.GetStatus()

		case <-timeout:
			orange.Outcome.Timeout()
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

	http.HandleFunc("/", handler)
	fmt.Println("Starting service...")

	cfg.Load()
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Panic(err)
		}
		log.SetOutput(file)
	}

	log.Println("Configuration load")
	point := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)
	if err := http.ListenAndServe(point, nil); err != nil {
		log.Panic(err)
	}

}
