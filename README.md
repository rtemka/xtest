### **Использование REST API**


```bash
# GET
curl "http://localhost:8080/api/latest"
# {"AMD":9315384.806843784,"AUD":32652.627516544842...}

# GET
curl "http://localhost:8080/api/btcusdt"
# {"timestamp":1658659428,"value":22514.1}

# пагинация и фильтр по дате
curl -X POST "http://localhost:8080/api/btcusdt?limit=10&offset=10&date=gte:2022-07-24T00:00:00"
# {"total":10,"history":[{"timestamp":1658659428,"value":22514.1}...]}

# GET
curl "http://localhost:8080/api/currencies"
# {"AMD":13.8929,"AUD":39.6347,"AZN":33.7598,...}

# фильтр по дате и валюте (пагинация тоже есть)
curl -X POST "http://localhost:8080/api/currencies?currency=HUF&date=2022-07-24"
# {"total":1,"history":[{"HUF":14.6643,"date":"2022-07-24"}]}
```