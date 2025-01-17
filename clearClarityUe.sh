#!/usr/bin/env bash
#
# This script is used to clear the setup generated by clarityUe.sh
#
# Author: Daniel Camps (daniel.camps@i2cat.net)
# Copyright: i2CAT


#############################
# Parsing inputs parameters
#############################

usage() { echo "Usage: $0 [-n <NUM_UEs>]" 1>&2; exit 1; }

while getopts ":n:" o; do
    case "${o}" in
        n)
            NUM_UES=${OPTARG}
            n=1
            echo "NUM_UEs="$NUM_UES
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [ -z "${n}" ]; then
    usage
fi

##############################
# Environment configuration
##############################

# Check OS
if [ -f /etc/os-release ]; then
    # freedesktop.org and systemd
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
else
    # Fall back to uname, e.g. "Linux <version>", also works for BSD, etc.
    OS=$(uname -s)
    VER=$(uname -r)
    echo "This Linux version is too old: $OS:$VER, we don't support!"
    exit 1
fi

sudo -v
if [ $? == 1 ]
then
    echo "Without root permission, you cannot run the test due to our test is using namespace"
    exit 1
fi

GOPATH=$HOME/go
if [ $OS == "Ubuntu" ]; then
    GOROOT=/usr/local/go
elif [ $OS == "Fedora" ]; then
    GOROOT=/usr/lib/golang
fi
PATH=$PATH:$GOPATH/bin:$GOROOT/bin


###################################
# Deleting UE and MPTCP namespaces
##################################

# Deleting per UE namespaces
sudo ip xfrm policy flush
sudo ip xfrm state flush

for i in $(seq 1 $NUM_UES)
do
  BRNAME="br"$i
  sudo ifconfig $BRNAME down
  sudo brctl delbr $BRNAME

  UENS="UEns_"$i
  EXEC_UENS="sudo ip netns exec ${UENS}"
  VETH_UE_BRIDGE="veth_ue_"$i"_"$BRNAME

  sudo ip link del $VETH_UE_BRIDGE
  ${EXEC_UENS} ip link del ipsec0
  sudo ip netns del ${UENS}
done


#Deleting MPTCP netns
MPTCPNS="MPTCPns"
EXEC_MPTCPNS="sudo ip netns exec ${MPTCPNS}"
sudo ip netns del ${MPTCPNS}
for i in $(seq 1 $NUM_UES)
do
  sudo ip link del "v_mph_"$i
  sudo ip link del "v_ueh_"$i
  sudo ip link set "brmptcp_"$i down
  sudo brctl delbr "brmptcp_"$i
done

# Killing any openvpn process
sudo killall openvpn
