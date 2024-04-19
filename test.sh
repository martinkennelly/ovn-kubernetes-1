#!/bin/bash

oc apply -f ~/artifacts/deployment/echoserver
oc delete -f ~/artifacts/deployment/echoserver
read -n 1 -p "input"
date
oc apply -f ~/artifacts/deployment/echoserver
time kubectl wait deployment -n default echoserverdeployment --for condition=Running=True --timeout=250s
date

