#!/bin/bash

function disable_systemd {
	systemctl stop stund || :
    systemctl disable stund
    rm -f /lib/systemd/system/stund.service
}

if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    if [[ "$1" = "0" ]]; then
	# Centrifugo is no longer installed, remove from init system
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
	    disable_systemd
	fi
	rm -f /etc/default/stund
    fi
elif [[ -f /etc/debian_version ]]; then
    # Debian/Ubuntu logic
    if [[ "$1" != "upgrade" ]]; then
	# Remove/purge
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
	    disable_systemd
	fi
	rm -f /etc/default/stund
    fi
fi