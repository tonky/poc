#!/bin/bash
mkdir -p $GOPATH/bin

export GOPATH="/home/ubuntu/go"
export PATH="$PATH:$GOPATH/bin"

echo "export GOPATH=$GOPATH" >> ~/.bashrc
echo "export PATH=$PATH:$GOPATH/bin" >> ~/.bashrc

echo "net.ipv4.tcp_tw_recycle = 1" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_tw_reuse = 1" | sudo tee -a /etc/sysctl.conf
sudo sysctl --system

# override docker's storage to use 'overlay2', since 'aufs' is too slow
sudo mkdir /etc/systemd/system/docker.service.d
sudo cp docker/systemd-override.conf /etc/systemd/system/docker.service.d/override.conf

# ################################################################################
# uncomment following section when using EC2 with SSD on /dev/xvdb mounted on /mnt
# ################################################################################

# sudo umount /mnt && sudo mkfs.ext4 -F /dev/xvdb && sudo mount /dev/xvdb /mnt
# sudo mkdir /mnt/docker && sudo chmod 777 /mnt/docker && ln -s /mnt/docker /var/lib

# ################################################################################

sudo add-apt-repository ppa:longsleep/golang-backports -y

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository -y "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

sudo curl -L "https://github.com/docker/compose/releases/download/1.13.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

sudo apt-get update -y
sudo apt-get install -y docker-ce golang-go apache2-utils wrk
sudo systemctl start docker

go get -u github.com/golang/dep/...
