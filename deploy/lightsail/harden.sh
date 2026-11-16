#!/usr/bin/env bash
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold full-upgrade
apt-get -y install unattended-upgrades fail2ban python3-systemd ca-certificates curl gnupg

cat >/etc/apt/apt.conf.d/20auto-upgrades <<'X'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
X
systemctl enable --now unattended-upgrades

if ! swapon --show | grep -q /swapfile; then
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
fi
grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >>/etc/fstab

cat >/etc/sysctl.d/99-hardening.conf <<'X'
vm.swappiness=10
net.ipv4.tcp_syncookies=1
net.ipv4.conf.all.accept_redirects=0
net.ipv4.conf.default.accept_redirects=0
net.ipv6.conf.all.accept_redirects=0
net.ipv6.conf.default.accept_redirects=0
net.ipv4.conf.all.send_redirects=0
net.ipv4.conf.default.send_redirects=0
net.ipv4.conf.all.accept_source_route=0
net.ipv6.conf.all.accept_source_route=0
net.ipv4.icmp_echo_ignore_broadcasts=1
net.ipv4.conf.all.log_martians=1
kernel.kptr_restrict=2
kernel.dmesg_restrict=1
fs.protected_hardlinks=1
fs.protected_symlinks=1
X
sysctl --system >/dev/null

cat >/etc/ssh/sshd_config.d/00-hardening.conf <<'X'
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
LoginGraceTime 30
X11Forwarding no
AllowAgentForwarding no
AllowUsers ubuntu
ClientAliveInterval 300
ClientAliveCountMax 2
X
sshd -t
systemctl try-reload-or-restart ssh

cat >/etc/fail2ban/jail.local <<'X'
[sshd]
enabled = true
backend = systemd
maxretry = 5
findtime = 10m
bantime = 1h
X
systemctl enable fail2ban
systemctl restart fail2ban

install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" >/etc/apt/sources.list.d/docker.list
apt-get update
apt-get -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

mkdir -p /etc/docker
cat >/etc/docker/daemon.json <<'X'
{
  "log-driver": "local",
  "log-opts": { "max-size": "10m", "max-file": "3" },
  "live-restore": true
}
X
systemctl enable docker
systemctl restart docker
usermod -aG docker ubuntu

echo "=== RESUMEN ==="
lsb_release -ds
free -h
swapon --show
sshd -T | grep -Ei '^(permitrootlogin|passwordauthentication|maxauthtries|allowusers) '
fail2ban-client status sshd | tail -3
docker --version
docker compose version
systemctl is-active unattended-upgrades fail2ban docker
[ -f /var/run/reboot-required ] && echo "REBOOT REQUIRED" || echo "no reboot required"
