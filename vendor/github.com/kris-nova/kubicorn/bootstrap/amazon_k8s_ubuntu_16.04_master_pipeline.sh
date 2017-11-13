#!/usr/bin/env bash
set -e
cd ~

# ------------------------------------------------------------------------------------------------------------------------

### Workaround for different os.Hostname and amazon API hostname
hostname -f > /etc/hostname
hostnamectl set-hostname $(hostname -f)

# curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
# touch /etc/apt/sources.list.d/kubernetes.list
# sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

#apt-get update -y
#apt-get install -y \
#      jq
#     socat \
#     ebtables \
#     docker.io \
#     apt-transport-https \
#     kubelet \
#     kubeadm=1.7.0-00 \
#     cloud-utils

sed -i -e 's|Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"|Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"\nEnvironment="KUBELET_EXTRA_ARGS=--cloud-provider=aws"|' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

# curl https://storage.googleapis.com/kubernetes-helm/helm-v2.6.0-linux-amd64.tar.gz | tar xz --strip 1 -C /usr/bin/

# systemctl enable docker
# systemctl start docker
TOKEN=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDTOKEN')
PORT=$(cat /etc/kubicorn/cluster.json | jq -r '.values.itemMap.INJECTEDPORT | tonumber')
PUBLICIP=$(ec2metadata --public-ipv4 | cut -d " " -f 2)
PRIVATEIP=$(ip addr show dev eth0 | awk '/inet / {print $2}' | cut -d"/" -f1)

kubeadm reset
kubeadm init --apiserver-bind-port ${PORT} --token ${TOKEN}  --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans ${PUBLICIP} ${PRIVATEIP}

sed -i -e 's|    - --address=127.0.0.1|    - --address=127.0.0.1\n    - --cloud-provider=aws\n    - --attach-detach-reconcile-sync-period=1m0s|' /etc/kubernetes/manifests/kube-controller-manager.yaml

until kubectl get pod --kubeconfig /etc/kubernetes/admin.conf; do
    echo "Waiting for Kubernetes ready"
    sleep 1
done

cat <<EOF | kubectl --kubeconfig /etc/kubernetes/admin.conf create -f -
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1
metadata:
  name: standard
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
EOF

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

helm install /opt/helm/pipeline-cluster
