import time
import requests
import json 
import uuid


data = {
    "Events": [
        "24865",
        "24866",
        "24867",
        "24868",
        "24869"
    ],
    "Imei": "861715031832588",
    "Inn": "1234567890",
    "Id": None,
    "ConfigFR": {
            "Host": "ferma-test.ofd.ru",
            "Login": "efir_fa_test",
            "Password": "Qk3620190Qq"
    },
    "InQueue": 0,
    "Fields": {
        "Request": {
            "Inn": "1234567890",
            "Type": "Income",
            "InvoiceId": None,
            "LocalDate": None,
            "CustomerReceipt": {
                    "TaxationSystem": "Common",
                    "Email": "beieni390@mail.ru",
                    "Phone": "+79301529765",
                    "PaymentType": 1,
                    "AutomatNumber": 2,
                    "KktFA": True,
                    "BillAddress": "Point Adress",
                    "Items": [
                        {
                            "PaymentMethod": 4,
                            "PaymentType": 0,
                            "Quantity": 5,
                            "Price": 1,
                            "Amount": 5,
                            "Label": "Тесточино",
                            "Vat": "Vat20"
                        }
                    ],
                "PaymentItems": [
                        {
                            "PaymentType": 0,
                            "Sum": 5
                        }
                    ]
            }
        }
    }
}

url = 'http://127.0.0.1:9010'


def generate_id():
    return uuid.uuid4().hex


if __name__ == "__main__":

    check_id = generate_id()

    try:
        data['Id'] = check_id
        data['Fields']['Request']['InvoiceId'] = check_id
        data['Fields']['Request']['LocalDate'] = time.strftime(
            "%Y-%m-%d %H:%M:%S", time.localtime())
        print('Fiskal ID# {}'.format(check_id))
        response = requests.post(url, json.dumps(data))
        print(response.text)
        
    except requests.exceptions.HTTPError as err:
        print('Fail:', err )                

    



