#!/usr/bin/env bash
set -e
cd ~

#------------------------------------------------------------------------------------------------

#curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
#touch /etc/apt/sources.list.d/kubernetes.list
#sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
#
apt-get update -y
apt-get install -y \
     jq
#    socat \
#    ebtables \
#    docker.io \
#    apt-transport-https \
#    kubelet \
#    kubeadm=1.7.0-00
#
#systemctl enable docker
#systemctl start docker

TOKEN=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDTOKEN')
MASTER=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDMASTER')

kubeadm reset
kubeadm join --token ${TOKEN} ${MASTER}
