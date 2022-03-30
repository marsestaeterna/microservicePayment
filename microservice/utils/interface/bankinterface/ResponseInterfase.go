package bankinterface

import (

)

type Response struct {
    Success bool
    ActionCode int
    ErrorCode int
    ErrorMessage string
    OrderStatus  int
    Error struct {
        Code int
        Description string
        Message string
    }
    BankType int
    MerchantId string
    GateWay string
    Sum int
    Status int
    Code int
    Data map[string]interface{}
}


