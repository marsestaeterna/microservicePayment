package bank

import (
    sberBank "bank/sber"
    interfaceBank "interface/bankinterface"
)
// instance of type banks
var bankSber sberBank.NewSberStruct
// end of instanse type banks
var ArrayBanks = map[string] interfaceBank.Bank{
  "Sber": bankSber.NewBank(),
}
func GetBank(bankName string) (interfaceBank.Bank) {
  for key,element := range ArrayBanks {
    if key == bankName {
      return element
    }
  }
  return nil
}



