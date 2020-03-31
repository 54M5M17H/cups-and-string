#!/bin/bash



if [ "$#" -ne 5 ]; then
    echo "Expects 6 arguments: user, node ip, node number, path to the executable, and node ip lookup path"
    exit 1
fi

USER=$1
NODE_IP=$2
NODE_NUM=$3
PATH_TO_OVERLAY_EXEC=$4
NODE_IP_LOOKUP_PATH=$5

TMP_OVERLAY_UNIT="/tmp/overlay_unit_file"
SERVICE="/etc/systemd/system/overlay.service"

echo "[Unit] 
AssertPathExists=$PATH_TO_OVERLAY_EXEC

[Service] 
ExecStart=$PATH_TO_OVERLAY_EXEC
Environment=\"NODE_NUM=$NODE_NUM\"
Environment=\"NODE_IP_LOOKUP_PATH=$NODE_IP_LOOKUP_PATH\"
Restart=no 
Type=simple 

[Install] 
WantedBy=multi-user.target 
" > $TMP_OVERLAY_UNIT

echo "Warning: May require your password twice to copy file"

scp $TMP_OVERLAY_UNIT $USER@$NODE_IP:$TMP_OVERLAY_UNIT
# separate because we can't sudo for remote scp
ssh -t $USER@$NODE_IP "sudo mv $TMP_OVERLAY_UNIT $SERVICE && sudo systemctl daemon-reload"
