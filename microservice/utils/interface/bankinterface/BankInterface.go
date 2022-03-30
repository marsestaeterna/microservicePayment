package bankinterface

import (
    connectionPostgresql "connectionDB/connect"
)

type Bank interface {
    CreateOrder()
    GetStatusOrder()
    CancelOrder()
    GetPaymentType()
    InitBankData(*Request,*connectionPostgresql.DatabaseInstance)
    Timeout()
}