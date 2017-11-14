#!/usr/bin/env bash
set -e
cd ~

#------------------------------------------------------------------------------------------------
hostname -f > /etc/hostname
hostnamectl set-hostname $(hostname -f)

#curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
#touch /etc/apt/sources.list.d/kubernetes.list
#sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
#
#apt-get update -y
#apt-get install -y \
#     jq
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

sed -i -e 's|Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"|Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"\nEnvironment="KUBELET_EXTRA_ARGS=--cloud-provider=aws"|' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

systemctl daemon-reload
systemctl restart kubelet.service

export KUBECONFIG=/etc/kubernetes/kubelet.conf

until kubectl get node
do
  echo "Waiting...."
  kubeadm reset
  kubeadm join --token ${TOKEN} ${MASTER}
  sleep 10
done
