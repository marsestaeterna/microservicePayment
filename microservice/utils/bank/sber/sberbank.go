package sber

import(
    "bytes"
	"encoding/json"
    "fmt"
    "net/http"
    "io/ioutil"
	"runtime"
    "log"
    randString "randgeneratestring"
    interfaceBank "interface/bankinterface"
    connectionPostgresql "connectionDB/connect"
)
const TransactionState_Idle 				= 0; // Transaction Idle
const TransactionState_MoneyHoldStart 		= 1; // создали транзакцию банка
const TransactionState_MoneyHoldWait 		= 2; // ожидает ответ от банка
const TransactionState_MoneyDebitStart		= 8;
const TransactionState_MoneyDebitWait		= 9;
const TransactionState_MoneyDebitOk			= 10;
const TransactionState_Error 				= 120;


const Order_Registered = 0 //- заказ зарегистрирован, но не оплачен;
const Order_HoldMoney  = 1 //- предавторизованная сумма удержана (для двухстадийных платежей);
const Order_FullAuthorizationOfTheAmount =  2 //- проведена полная авторизация суммы заказа;
const Order_AuthorizationCanceled = 3 //- авторизация отменена;
const Order_RefundOperationPerformed  = 4 //- по транзакции была проведена операция возврата;
const Order_AuthorizationThroughTheServerHasBeenInitiated = 5 //- инициирована авторизация через сервер контроля доступа банка-эмитента;
const Order_AuthorizationDenied = 6 //- авторизация отклонена.

type Sber struct {
    Name string
    Counter int
    PaymentType int
    UrlCreateOrder string
    UrlGetStatusOrder string
    UrlCancelOrder string
    Req interfaceBank.Request
    Res interfaceBank.Response
}

type NewSberStruct struct {
    Sber
}

func (s *Sber) Finish() {
	runtime.Goexit()
}

func (s *Sber) MakeEventAndLog() {
    defer s.Finish()
    parametrs := make(map[string]interface{})
    Where := make(map[string]interface{})
    Where["id"] = s.Req.IdTransaction
    parametrs["ps_desc"] = s.Res.Error.Description
    parametrs["error"] = s.Res.Error.Message
    if s.Res.Data["orderId"] == nil {
         parametrs["ps_order"] = "no order_id"
    }else {
         parametrs["ps_order"] = s.Res.Data["orderId"].(string)
    }
    
    parametrs["status"] = s.Res.Code
    log.Printf("%+v",parametrs)
    connectDb.Set("transaction", parametrs, Where)
}

func (s *Sber) MakeJsonRequestDepositOrder(sum int) ([]byte,error){
    requestOrder := make(map[string]interface{})
    requestOrder["amount"] = sum
    requestOrder["orderId"] = s.Res.Data["orderId"]
    data, err := json.Marshal(requestOrder)
    if err != nil {
        log.Printf("%+v",err)
        return nil, err
    }else {
        return data ,nil
    }
}

func (s *Sber) MakeJsonRequestStatusOrder() ([]byte,error){
    requestOrder := make(map[string]interface{})
    requestOrder["token"] = s.Req.PaymentToken
    requestOrder["orderId"] = s.Res.Data["orderId"]
    data, err := json.Marshal(requestOrder)
    if err != nil {
        log.Printf("%+v",err)
        return nil, err
    }else {
        return data ,nil
    }
}

func (s *Sber) MakeJsonOrderRequestCreateOrder() ([]byte, error){
    var orderString randString.GenerateString
    orderString.RandStringRunes()
    orderNumber := orderString.String
    requestOrder := make(map[string]interface{})
    requestOrder["merchant"] = s.Req.MerchantId
    requestOrder["orderNumber"] = orderNumber
    requestOrder["language"] = s.Req.Config.Language
    requestOrder["preAuth"] = true
    requestOrder["description"] = s.Req.Config.Description
    requestOrder["paymentToken"] = s.Req.PaymentToken
    requestOrder["amount"] = s.Req.Sum
    requestOrder["currencyCode"] = s.Req.Config.CurrensyCode
    requestOrder["returnUrl"]  = "https://test.ru" 
   
    data, err := json.Marshal(requestOrder)
    if err != nil {
        log.Printf("%+v",err)
        return nil, err
    }else {
        return data ,nil
    }
}

func (s *Sber) GetInfoRunnigCounter(w http.ResponseWriter){
    fmt.Fprintf(w, "%s: Running\n", s.Name)
}

func (s *Sber) InitBankData(data *interfaceBank.Request,Db *connectionPostgresql.DatabaseInstance){
    s.Req = *data
    connectDb = *Db
}

func (s *Sber) GetResponse() *interfaceBank.Response {
    return &s.Res
}

func (s *Sber) CreateOrder(){
     dataPush,err := s.MakeJsonOrderRequestCreateOrder()
     if err != nil {
         s.Res.Code = TransactionState_Error
         s.Res.Error.Message = fmt.Sprintf("%s",err)
         s.Res.Error.Description = "ошибка преобразования map[string]interface{} в json (Создание транзакции)"
         s.MakeEventAndLog()
     }
     s.Call("POST",s.UrlCreateOrder,dataPush)
    if s.Res.Success != true {
        s.Res.Code = TransactionState_Error
        s.MakeEventAndLog()
    }else {
        s.Res.Code = TransactionState_MoneyHoldWait
        s.Res.Error.Description = "Ожидаем оплаты"
        s.MakeEventAndLog()
        s.GetStatusOrder()
    }
}


func (s *Sber) GetStatusOrder(){
    dataPush,err := s.MakeJsonRequestStatusOrder()
     if err != nil {
         s.Res.Code = TransactionState_Error
         s.Res.Error.Message = fmt.Sprintf("%s",err)
         s.Res.Error.Description = "ошибка преобразования map[string]interface{} в json (Опрос статуса транзакции)"
         s.MakeEventAndLog()
     }
    s.Call("POST",s.UrlGetStatusOrder,dataPush)
    if s.Res.ActionCode != 0 {
        s.Res.Error.Message = s.Res.ErrorMessage
        s.Res.Error.Description = s.Res.ErrorMessage
        s.Res.Code = TransactionState_Error
        s.MakeEventAndLog()
    }else {
        if s.Req.Config.PayType == 1 {
            if s.Res.ActionCode == Order_HoldMoney {
                s.Res.Code = Order_HoldMoney
                s.Res.Error.Description = fmt.Sprintf("Сумма %v удержана, ожидайте завершения транзакции",string(s.Req.Sum))
                s.Res.Error.Message = fmt.Sprintf("Сумма %v удержана, ожидайте завершения транзакции",string(s.Req.Sum))
                s.MakeEventAndLog()
            }else {
                s.Res.Code = TransactionState_Error
                s.Res.Error.Description = s.Res.ErrorMessage
                s.Res.Error.Message = s.Res.ErrorMessage
                s.MakeEventAndLog()
            }
        }
    }
}

func (s *Sber) CancelOrder(){

    if s.Req.Config.PayType == 1 {
        dataPush,err := s.MakeJsonRequestDepositOrder(s.Req.Sum)
         if err != nil {
         s.Res.Code = TransactionState_Error
         s.Res.Error.Message = fmt.Sprintf("%s",err)
         s.Res.Error.Description = "ошибка преобразования map[string]interface{} в json (Списание денег транзакции)"
         s.MakeEventAndLog()
     }
        s.Call("POST",s.UrlCancelOrder,dataPush)
        if s.Res.ErrorCode !=0 {
            s.Res.Code = TransactionState_Error
            s.Res.Error.Message = s.Res.ErrorMessage
            s.Res.Error.Description = s.Res.ErrorMessage
            s.MakeEventAndLog()
        }else {
            s.Res.Code = TransactionState_MoneyDebitOk
            s.Res.Error.Message =  s.Res.ErrorMessage
            s.Res.Error.Description = "Деньги списаны"
        }
    }else {

    }
}

func (s *Sber) Timeout(){
    s.Res.Code = TransactionState_Error
	s.Res.Error.Message = fmt.Sprintf("Cancelled by a Timeout of %s", "Сбербанк")
    s.Res.Error.Description = "Нет ответа от банка"
    s.MakeEventAndLog()
}

func (s *Sber) GetPaymentType(){

}

func (s *Sber) Call(method string, url string, json_request []byte) {
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(json_request))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		s.Res.Code = 0
		s.Res.Status = TransactionState_Error
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal([]byte(body), &s.Res)
}
var connectDb connectionPostgresql.DatabaseInstance
func (sber *NewSberStruct) NewBank() interfaceBank.Bank  /* указатель с типом interfaceBank.Bank*/ {
    return &NewSberStruct{
        Sber: Sber{
        Name: "Sber",
        Counter: 0,
        PaymentType: 1, // srandart type payment 
        UrlCreateOrder: "https://3dsec.sberbank.ru/payment/google/payment.do",
        UrlGetStatusOrder: "https://3dsec.sberbank.ru/payment/google/getOrderStatusExtended.do",
        UrlCancelOrder: "https://3dsec.sberbank.ru/payment/google/deposit.do", // use for 
       },
    }
}