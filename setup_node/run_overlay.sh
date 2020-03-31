#!/bin/bash

if [ "$#" -ne 6 ]; then
    echo "Expects 5 arguments: machine username, machine IP, path to IP local lookup file, and destination path for IP lookup on remote machine, and path to overlay bin, and local path to overlay dir"
    exit 1
fi

USER=$1
IP=$2
LOCAL_IP_LOOKUP_PATH=$3
REMOTE_IP_LOOKUP_PATH=$4
PATH_TO_OVERLAY_BIN=$5
LOCAL_PATH_TO_OVERLAY_DIR=$6

echo "Copying over IP lookup"
scp $LOCAL_IP_LOOKUP_PATH ${USER}@${IP}:${REMOTE_IP_LOOKUP_PATH}

echo "Stopping overlay service"
ssh ${USER}@${IP} 'sudo systemctl stop overlay'

echo "Installing overlay..."
# copy from host
# will overwrite 

scp -r $LOCAL_PATH_TO_OVERLAY_DIR ${USER}@${IP}:/tmp/

echo "Building binary"
# go install & restart daemon
ssh ${USER}@${IP} "cd /tmp/overlay && /usr/local/go/bin/go build -o $PATH_TO_OVERLAY_BIN"

echo 'Installed. Running overlay...'

ssh -tt ${USER}@${IP} 'sudo systemctl start overlay'
