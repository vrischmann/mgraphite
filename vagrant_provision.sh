#!/bin/bash

export DEBIAN_FRONTEND=noninteractive

apt-get -qq update
apt-get -qq install curl apt-transport-https

cd /tmp && rm *.deb

curl -s -LOJ https://grafanarel.s3.amazonaws.com/builds/grafana_3.0.2-1463383025_amd64.deb
dpkg -i grafana_3.0.2-1463383025_amd64.deb

curl -s -LOJ https://github.com/lomik/go-carbon/releases/download/v0.7.1/go-carbon_0.7.1_amd64.deb
dpkg -i go-carbon_0.7.1_amd64.deb

apt-get -qq install -f

cat <<EOF > /etc/carbon.conf
[common]
logfile = "/var/log/go-carbon/go-carbon.log"
log-level = "info"
graph-prefix = "carbon.agents.{host}."
metric-interval = "1m0s"
max-cpu = 1

[whisper]
data-dir = "/tmp/data/graphite/whisper/"
schemas-file = "/etc/storage-schema.conf"
aggregation-file = ""
workers = 1
max-updates-per-second = 0
enabled = true

[cache]
max-size = 1000000
input-buffer = 51200

[udp]
enabled = false

[tcp]
listen = ":2003"
enabled = true

[pickle]
enabled = false

[carbonlink]
listen = "127.0.0.1:7002"
enabled = true
read-timeout = "30s"
query-timeout = "100ms"
EOF

cat <<EOF > /etc/storage-schema.conf.conf
[foobar]
pattern = .*
retentions = 1m;60d
EOF
