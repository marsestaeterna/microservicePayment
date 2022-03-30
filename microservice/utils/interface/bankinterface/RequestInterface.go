package bankinterface

import (

)

type Request struct {
    Config struct {
        BankType int
        PayType int
        CurrensyCode int
        Language string
        Description string
    }
    IdTransaction string
    MerchantId string
    GateWay string
    PaymentToken string
    Sum int
}