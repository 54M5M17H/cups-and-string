#!/bin/bash

echo "RUNNING"

# needs 2 arguments: username and IP
# $# is arguments length

if [ "$#" -ne 2 ]; then
    echo "Expects 2 arguments: machine username and machine IP"
    exit 1
fi

USER=$1
echo $USER
IP=$2
echo $IP

SUDOER_LOC="/etc/sudoers.d/$USER"
TMP_SUDOER_FILE="/tmp/sudoer_file"

echo "
%$USER ALL= NOPASSWD: /bin/systemctl start overlay.service
%$USER ALL= NOPASSWD: /bin/systemctl start overlay
%$USER ALL= NOPASSWD: /bin/systemctl restart overlay.service
%$USER ALL= NOPASSWD: /bin/systemctl restart overlay
%$USER ALL= NOPASSWD: /bin/systemctl stop overlay.service
%$USER ALL= NOPASSWD: /bin/systemctl stop overlay
" > $TMP_SUDOER_FILE

echo "Warning: Your password may be required twice to copy a file to remote machine."
scp $TMP_SUDOER_FILE $USER@$IP:$TMP_SUDOER_FILE
# separate because we can't sudo for remote scp
ssh -t $USER@$IP "sudo cat $TMP_SUDOER_FILE > $SUDOER_LOC && sudo pkexec chown root:root /etc/sudoers /etc/sudoers.d -R"
# -t allows us to ask for password
# final part of command sets owner of sudoers back to root -- bit dodgy when done by ssh
