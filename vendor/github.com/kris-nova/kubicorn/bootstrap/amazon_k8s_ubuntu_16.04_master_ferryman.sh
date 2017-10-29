#!/usr/bin/env bash
set -e
cd ~

# ------------------------------------------------------------------------------------------------------------------------

# curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
# touch /etc/apt/sources.list.d/kubernetes.list
# sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
      jq
#     socat \
#     ebtables \
#     docker.io \
#     apt-transport-https \
#     kubelet \
#     kubeadm=1.7.0-00 \
#     cloud-utils

# curl https://storage.googleapis.com/kubernetes-helm/helm-v2.6.0-linux-amd64.tar.gz | tar xz --strip 1 -C /usr/bin/

# systemctl enable docker
# systemctl start docker
TOKEN=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDTOKEN')
PORT=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDPORT | tonumber')
PUBLICIP=$(ec2metadata --public-ipv4 | cut -d " " -f 2)
PRIVATEIP=$(ip addr show dev eth0 | awk '/inet / {print $2}' | cut -d"/" -f1)

kubeadm reset
kubeadm init --apiserver-bind-port ${PORT} --token ${TOKEN}  --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans ${PUBLICIP} ${PRIVATEIP}

# Thanks Kelsey :)
kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')" --kubeconfig /etc/kubernetes/admin.conf


export KUBECONFIG=/etc/kubernetes/admin.conf

kubectl create serviceaccount --namespace kube-system tiller --kubeconfig /etc/kubernetes/admin.conf
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller --kubeconfig /etc/kubernetes/admin.conf
helm init --service-account tiller

mkdir -p /home/ubuntu/.kube
cp /etc/kubernetes/admin.conf /home/ubuntu/.kube/config
chown -R ubuntu:ubuntu /home/ubuntu/.kube

until helm list
do
  echo "Waiting...."
  kubectl get po --all-namespaces
  sleep 5
done

helm install /opt/helm/cluster-ingress-controller-chart
helm install /opt/helm/cluster-prometheus-chart
