# Introduction
This document introduces the nordicenergy's package release using standard packaging system, RPM and Deb packages.

Standard packaging system has many benefits, like extensive tooling, documentation, portability, and complete design to handle different situation.

# Package Content
The RPM/Deb packages will install the following files/binary in your system.
* /usr/sbin/nordicenergy
* /usr/sbin/nordicenergy-setup.sh
* /usr/sbin/nordicenergy-rclnet.sh
* /etc/nordicenergy/nordicenergy.conf
* /etc/nordicenergy/rclnet.conf
* /etc/systemd/system/nordicenergy.service
* /etc/sysctl.d/99-nordicenergy.conf

The package will create `nordicenergy` group and `nordicenergy` user on your system.
The nordicenergy process will be run as `nordicenergy` user.
The default blockchain DBs are stored in `/home/nordicenergy/nordicenergy_db_?` directory.
The configuration of nordicenergy process is in `/etc/nordicenergy/nordicenergy.conf`.

# Package Manager
Please take sometime to learn about the package managers used on Fedora/Debian based distributions.
There are many other package managers can be used to manage rpm/deb packages like [Apt]<https://en.wikipedia.org/wiki/APT_(software)>,
or [Yum]<https://www.redhat.com/sysadmin/how-manage-packages>

# Setup customized repo
You just need to do the setup of nordicenergy repo once on a new host.
**TODO**: the repo in this document are for development/testing purpose only.

Official production repo will be different.

## RPM Package
RPM is for Redhat/Fedora based Linux distributions, such as Amazon Linux and CentOS.

```bash
# do the following once to add the nordicenergy development repo
curl -LsSf http://haochen-nordicenergy-pub.s3.amazonaws.com/pub/yum/nordicenergy-dev.repo | sudo tee -a /etc/yum.repos.d/nordicenergy-dev.repo
sudo rpm --import https://raw.githubusercontent.com/nordicenergy/nordicenergy-open/master/nordicenergy-release/nordicenergy-pub.key
```

## Deb Package
Deb is supported on Debian based Linux distributions, such as Ubuntu, MX Linux.

```bash
# do the following once to add the nordicenergy development repo
curl -LsSf https://raw.githubusercontent.com/nordicenergy/nordicenergy-open/master/nordicenergy-release/nordicenergy-pub.key | sudo apt-key add
echo "deb http://haochen-nordicenergy-pub.s3.amazonaws.com/pub/repo bionic main" | sudo tee -a /etc/apt/sources.list

```

# Test cases
## installation
```
# debian/ubuntu
sudo apt-get update
sudo apt-get install nordicenergy

# fedora/amazon linux
sudo yum install nordicenergy
```
## configure/start
```
# dpkg-reconfigure nordicenergy (TODO)
sudo systemctl start nordicenergy
```

## uninstall
```
# debian/ubuntu
sudo apt-get remove nordicenergy

# fedora/amazon linux
sudo yum remove nordicenergy
```

## upgrade
```bash
# debian/ubuntu
sudo apt-get update
sudo apt-get upgrade

# fedora/amazon linux
sudo yum update --refresh
```

## reinstall
```bash
remove and install
```

# Rclnet
## install latest rclnet
```bash
# debian/ubuntu
curl -LO https://downloads.rclnet.org/v1.52.3/rclnet-v1.52.3-linux-amd64.deb
sudo dpkg -i rclnet-v1.52.3-linux-amd64.deb

# fedora/amazon linux
curl -LO https://downloads.rclnet.org/v1.52.3/rclnet-v1.52.3-linux-amd64.rpm
sudo rpm -ivh rclnet-v1.52.3-linux-amd64.rpm
```

## do rclnet
```bash
# validator runs on shard1
sudo -u nordicenergy nordicenergy-rclnet.sh /home/nordicenergy 0
sudo -u nordicenergy nordicenergy-rclnet.sh /home/nordicenergy 1

# explorer node
sudo -u nordicenergy nordicenergy-rclnet.sh -a /home/nordicenergy 0
```

# Setup explorer (non-validating) node
To setup an explorer node (non-validating) node, please run the `nordicenergy-setup.sh` at first.

```bash
sudo /usr/sbin/nordicenergy-setup.sh -t explorer -s 0
```
to setup the node as an explorer node w/o blskey setup.

# Setup new validator
Please copy your blskey to `/home/nordicenergy/.ngy/blskeys` directory, and start the node.
The default configuration is for validators on mainnet. No need to run `nordicenergy-setup.sh` script.

# Start/stop node
* `systemctl start nordicenergy` to start node
* `systemctl stop nordicenergy` to stop node
* `systemctl status nordicenergy` to check status of node

# Change node configuration
The node configuration file is in `/etc/nordicenergy/nordicenergy.conf`.  Please edit the file as you needed.
```bash
sudo vim /etc/nordicenergy/nordicenergy.conf
```

# Support
Please open new github issues in https://github.com/nordicenergy/nordicenergy-core/issues.
