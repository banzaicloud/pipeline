#!/usr/bin/env bash
set -e
cd ~

# ------------------------------------------------------------------------------------------------------------------------
# These values are injected into the script. We are explicitly not using a templating language to inject the values
# as to encourage the user to limit their use of templating logic in these files. By design all injected values should
# be able to be set at runtime, and the shell script real work. If you need conditional logic, write it in bash
# or make another shell script.
#
#
OPENVPN_CONF="INJECTEDCONF"
# ------------------------------------------------------------------------------------------------------------------------

PRIVATE_IP=$(curl http://169.254.169.254/metadata/v1/interfaces/private/0/ipv4/address)

# OpenVPN

apt-get update
apt-get install -y openvpn

echo -e ${OPENVPN_CONF} > /etc/openvpn/clients.conf

systemctl start openvpn@clients
systemctl enable openvpn@clients
