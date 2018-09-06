#!/bin/bash

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout default.key -out default.crt -subj "/CN=NGINXIngressController"
cat default.key default.crt > default.pem 
rm default.key default.crt