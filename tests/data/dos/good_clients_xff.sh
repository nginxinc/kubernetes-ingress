#!/usr/bin/env bash

HOST=$1
IP_AND_PORT=$2
URI='good_path.html'

declare -a array=(
    "/faq.php?sid=e6106109db52fec6127b73b152224dea#f4r0"
    "/styles/prosilver/theme/print.css"
    "/styles/prosilver/theme/images/bg_header.gif"
    "/styles/prosilver/template/styleswitcher.js"
)

while true; do
    URI=${array[$((RANDOM % 3))]}
    echo ${HOST}
    echo ${IP_AND_PORT}
    echo 'curl -b cookiefile -c cookiefile -L -s -o /dev/null -w "%{http_code}\n"  -m 10 --connect-timeout 5 -A "Mozilla/5.0 (iPhone; U; CPU iPhone OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5" -H "Host: ${HOST}" -H "X-Forwarded-For: 3.3.3.2" ${IP_AND_PORT}${URI}'
    curl -b cookiefile -c cookiefile -L -s -o /dev/null -w "%{http_code}\n" -m 10 --connect-timeout 5 -A "Mozilla/5.0 (iPhone; U; CPU iPhone OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5" -H "Host: ${HOST}" -H "X-Forwarded-For: 3.3.3.2" ${IP_AND_PORT}${URI} &
    URI=${array[$((RANDOM % 3))]}
    curl -b cookiefile -c cookiefile -L -s -o /dev/null -w "%{http_code}\n" -m 10 --connect-timeout 5 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_6_8) AppleWebKit/534.30 (KHTML, like Gecko) Chrome/12.0.742.112 Safari/534.30" -H "Host: ${HOST}" -H "X-Forwarded-For: 31.212.17.19" ${IP_AND_PORT}${URI} &
    URI=${array[$((RANDOM % 3))]}
    curl -b cookiefile -c cookiefile -L -s -o /dev/null -w "%{http_code}\n" -m 10 --connect-timeout 5 -A "Mozilla/5.0 (Windows; U; Windows NT 5.1; de; rv:1.9.2.3) Gecko/20100401 Firefox/3.6.3" -H "Host: ${HOST}" -H "X-Forwarded-For: 46.117.23.157" ${IP_AND_PORT}${URI} &

    curl -b cookiefile -c cookiefile -L -s -o /dev/null -w "%{http_code}\n" -m 10 --connect-timeout 5 -A "Mozilla/5.0 (Windows; U; Windows NT 5.1; de; rv:1.9.2.3) Gecko/20100401 Firefox/3.6.3" -H "Host: ${HOST}" -H "X-Forwarded-For: 111.212.17.19" ${IP_AND_PORT}

    killall curl
    sleep 0.2
done
