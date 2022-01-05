curl -X GET \
  'http://arcadia-api.gke.nginx.rocks/trading/rest/buy_stocks.php' \
  -H 'Accept: */*' \
  -H 'User-Agent: Thunder Client (https://www.thunderclient.io)' \
  -H 'Content-Type: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
	"trans_value":12,
	"qty":16,
	"company":"MSFT",
	"action":"buy",
	"stock_price":198
}'