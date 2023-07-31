#!/bin/bash

set -e

BASHRC=/etc/profile.d/climc.sh

grep -q rcadmin $BASHRC || echo 'source /etc/yunion/rcadmin' >> $BASHRC

source $BASHRC

climc sshkeypair-inject --admin --target-dir /root/

mkdir -p /etc/dropbear

mkdir -p /var/run/dropbear

test -f /etc/dropbear/dropbear_rsa_host_key || dropbearkey -t rsa -f /etc/dropbear/dropbear_rsa_host_key

dropbear -RFEjk -G root -p 22
