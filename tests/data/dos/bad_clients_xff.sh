#!/usr/bin/env bash

HOST=$1
IP_AND_PORT=$2
URI='bad_path.html'
NUM=600
CONNS=300
while true; do
    echo "ab -l -n ${NUM} -c ${CONNS} -d -s 5 \
        -H "Host: ${HOST}" \
        -H "X-Forwarded-For: 1.1.1.1" \
        ${IP_AND_PORT}${URI}"

    ab -l -n ${NUM} -c ${CONNS} -d -s 5 \
        -H "Host: ${HOST}" \
        -H "X-Forwarded-For: 1.1.1.1" \
        ${IP_AND_PORT}${URI} &

    ab -l -n ${NUM} -c ${CONNS} -d -s 5 \
        -H "Host: ${HOST}" \
        -H "X-Forwarded-For: 1.1.1.2" \
        ${IP_AND_PORT}${URI} &

    ab -l -n ${NUM} -c ${CONNS} -d -s 3 \
        -H "Host: ${HOST}" \
        -H "X-Forwarded-For: 1.1.1.3" \
        ${IP_AND_PORT}${URI}

    killall ab
done
