#!/bin/bash

BIN_DIR=/usr/bin
DATA_DIR=/var/lib/stund
SCRIPT_DIR=/usr/lib/stund/scripts
DUSER=stund

function install_systemd {
    cp -f $SCRIPT_DIR/stund.service /lib/systemd/system/stund.service
    systemctl enable stund
}

id stund &>/dev/null
if [[ $? -ne 0 ]]; then
    useradd --system -U -M $DUSER -s /bin/false -d $DATA_DIR
fi

if [ ! -d $DATA_DIR ]; then
    mkdir -p $DATA_DIR
fi

if [ ! -d $LOG_DIR ]; then
    mkdir -p $LOG_DIR
fi

chown -R -L $DUSER:$DUSER $DATA_DIR

# Add defaults file, if it doesn't exist
if [[ ! -f /etc/default/centrifugo ]]; then
    touch /etc/default/centrifugo
fi

# Distribution-specific logic
if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
	install_systemd
    fi
elif [[ -f /etc/debian_version ]]; then
    # Debian/Ubuntu logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
	install_systemd
    fi
fi