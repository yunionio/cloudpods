#!/bin/bash

set -e

BASHRC=/root/.bashrc

source $BASHRC

climc sshkeypair-inject --admin

mkdir -p /etc/dropbear

mkdir -p /var/run/dropbear

dropbearkey -t rsa -f /etc/dropbear/dropbear_rsa_host_key

dropbear -RFEjk -G root -p 22
