#!/bin/bash

echo "=== Cleanup Kubernetes Start ==="

# Reset kubeadm
kubeadm reset -f || true
systemctl stop kubelet || true

# Remove Kubernetes files
rm -rf /etc/kubernetes/
rm -rf /var/lib/etcd
rm -rf /var/lib/kubelet
rm -rf /etc/cni/
rm -rf /opt/cni/
rm -rf ~/.kube
rm -rf /root/.kube

# Install ipvsadm if not exist
if ! command -v ipvsadm >/dev/null 2>&1; then
    echo "Installing ipvsadm..."
    apt update
    apt install -y ipvsadm
fi

# Reset network
iptables -F
iptables -t nat -F
iptables -t mangle -F
iptables -X

ipvsadm --clear || true

# Remove containerd data
systemctl stop containerd || true

rm -rf /var/lib/containerd
rm -rf /etc/containerd

systemctl start containerd || true

# Remove Docker if exists
if systemctl list-unit-files | grep -q docker; then
    systemctl stop docker || true
    rm -rf /var/lib/docker
    systemctl start docker || true
fi

# Uninstall Kubernetes packages
apt purge -y kubeadm kubectl kubelet kubernetes-cni || true
apt autoremove -y || true

# Remove images cache
crictl rmi --prune || true

if command -v ctr >/dev/null 2>&1; then
    ctr -n k8s.io images rm $(ctr -n k8s.io images ls -q) || true
fi

echo "=== Cleanup Done ==="

echo "Reboot after 5 seconds..."
sleep 5

reboot



# Run
# chmod +x cleanup-k8s.sh
# sudo ./cleanup-k8s.sh

# Check
# ls /etc/kubernetes
# ss -lntp | grep 6443