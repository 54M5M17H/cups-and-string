#!/bin/bash

sudo mkdir -p /etc/overlay/
cp ./overlay /etc/overlay/overlay_svc
cp ./setup_node /etc/overlay/setup_node

cd ./cli
go build -o /usr/local/bin/cups-and-string
