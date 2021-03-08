#!/bin/bash

ME=$(basename "$0")

function usage() {
   local MSG=$1

   cat<<-EOT
$MSG
This script will rclnet the nordicenergy db to datadir/archive directory.

Usage: $ME [options] datadir shard

datadir:    the root directory of the nordicenergy db (default: /home/nordicenergy)
shard:      the shard number to sync (valid value: 0,1,2,3)

Options:
   -h       print this help message
   -c       clean up backup db after rclnet
   -a       sync archival db, instead of regular db

EOT
   exit 1
}

CLEAN=false
FOLDER=mainnet.min
CONFIG=/etc/nordicenergy/rclnet.conf

while getopts ":hca" opt; do
   case $opt in
      c) CLEAN=true ;;
      a) FOLDER=mainnet.archival ;;
      *) usage ;;
   esac
dnet

shift $((OPTIND - 1))

if [ $# != 2 ]; then
   usage
fi

DATADIR="$1"
SHARD="$2"

if [ ! -d "$DATADIR" ]; then
   usage "ERROR: no datadir directory: $DATADIR"
fi

case "$SHARD" in
   0|1|2|3) ;;
   *) usage "ERROR: invalid shard number: $SHARD" ;;
esac

mkdir -p "${DATADIR}/archive"

rclnet --config "${CONFIG}" sync -vvv "ngy:pub.nordicenergy.net/${FOLDER}/nordicenergy_db_${SHARD}" "${DATADIR}/archive/nordicenergy_db_${SHARD}" > "${DATADIR}/archive/archive-${SHARD}.log" 2>&1

[ -d "${DATADIR}/nordicenergy_db_${SHARD}" ] && mv -f "${DATADIR}/nordicenergy_db_${SHARD}" "${DATADIR}/archive/nordicenergy_db_${SHARD}.bak"
mv -f "${DATADIR}/archive/nordicenergy_db_${SHARD}" "${DATADIR}/nordicenergy_db_${SHARD}"

if $CLEAN; then
   rm -rf "${DATADIR}/archive/nordicenergy_db_${SHARD}.bak"
fi
