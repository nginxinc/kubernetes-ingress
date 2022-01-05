curl --location --request GET 'http://arcadia-api.gke.nginx.rocks/api/rest/execute_money_transfer.php' \
--header 'Content-Type: application/json' \
--data-raw '{
    "amount":92,
    "account":2075894,
    "currency":"GBP",
    "friend":"Vincent"
}'