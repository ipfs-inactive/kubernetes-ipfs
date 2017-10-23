#!/bin/bash
# Add repo
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

# Disable SELinux
setenforce 0
sed -i 's/^SELINUX=.*/SELINUX=disabled/g' /etc/selinux/config

# Install ZeroTier
curl -s https://install.zerotier.com/ | bash
systemctl enable zerotier-one && systemctl start zerotier-one
zerotier-cli join 93afae5963d7d91a

# Install dependencies
yum install -y kubelet kubeadm docker
systemctl enable docker && systemctl start docker
systemctl enable kubelet && systemctl start kubelet

# Join K8S
kubeadm join --token <token> 192.168.196.134:6443
