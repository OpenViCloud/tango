# Reset kubeadm

kubeadm reset -f
systemctl stop kubelet

# Remove all Kubernetes

rm -rf /etc/kubernetes/
rm -rf /var/lib/etcd
rm -rf /var/lib/kubelet
rm -rf /etc/cni/
rm -rf /opt/cni/
rm -rf ~/.kube

rm -rf /root/.kube

# Reset network rules. install if not exist: apt install ipvsadm -y

iptables -F
iptables -t nat -F
iptables -t mangle -F
iptables -X

ipvsadm --clear

# Delete container runtime data

systemctl stop containerd

rm -rf /var/lib/containerd
rm -rf /etc/containerd

systemctl start containerd

# Delete container runtime data if use docker

<!-- systemctl stop docker
rm -rf /var/lib/docker
systemctl start docker -->

# Uninstall package Kubernetes

apt purge -y kubeadm kubectl kubelet kubernetes-cni
apt autoremove -y

# Delete cache image Kubernetes

crictl rmi --prune
ctr -n k8s.io images rm $(ctr -n k8s.io images ls -q)

# Reboot

reboot

# Check kubeadm reset

ls /etc/kubernetes => No such file or directory
ss -lntp | grep 6443 => Not things found
