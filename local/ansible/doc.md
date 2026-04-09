mkdir -p ~/.kube
scp root@10.2.146.6:/etc/kubernetes/admin.conf ~/.kube/config
vi ~/.kube/config

# remote (tango production)

103.175.146.2

Master
103.175.146.6 -> private ip: 10.2.146.6

Worker
103.175.146.4 -> private ip: 10.2.146.4

    •	ansible → dùng để điều khiển VPS
    •	sshpass → nếu login bằng password
    •	git → để tải project hoặc playbook

# Install

sudo apt update
sudo apt install -y ansible sshpass git

mkdir k8s-lab
cd k8s-lab

# create ssh

ssh-keygen -t ed25519
ssh-copy-id root@10.2.146.6
ssh-copy-id root@10.2.146.4

# Test ping

ansible all -i inventory.ini -m ping

# Create yaml and install 1 Step

cd ~/k8s-lab
ansible-playbook -i inventory.ini k8s-install.yml
ansible-playbook -i inventory.ini master-init.yml
ansible-playbook -i inventory.ini worker-join.yml
