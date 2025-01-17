#!/bin/bash
# Jorge Navarro-Ortiz (jorgenavarro@ugr.es), University of Granada 2020

echo "Disabling interface towards Internet..."
sudo ifconfig eth0 down
echo "Changing IP addresses and forwarding..."
sudo ifconfig eth1 10.1.1.222/24
sudo ifconfig eth2 60.60.0.102/24
sudo sysctl -w net.ipv4.ip_forward=1
