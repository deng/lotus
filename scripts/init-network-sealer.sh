#!/usr/bin/env bash

set -xeo

SECTOR_SIZE=2KiB

sdt0111="${HOME}/.sector-0" 
sdt0222="${HOME}/.sector-1" 
sdt0333="${HOME}/.sector-2" 

ldt0111="${HOME}/.lotus-0" 
ldt0222="${HOME}/.lotus-1" 
ldt0333="${HOME}/.lotus-2" 

mdt0111="${HOME}/.lotussealer-t01000" 
mdt0222="${HOME}/.lotussealer-t01001" 
mdt0333="${HOME}/.lotussealer-t01002" 

ddt0111="${HOME}/.lotusdealer-t01000" 
ddt0222="${HOME}/.lotusdealer-t01001" 
ddt0333="${HOME}/.lotusdealer-t01002" 

rm -rf $mdt0111
rm -rf $mdt0222
rm -rf $mdt0333

rm -rf $ddt0111
rm -rf $ddt0222
rm -rf $ddt0333

env LOTUS_PATH="${ldt0222}" LOTUS_SEALER_PATH="${mdt0222}" ./lotus-sealer init --sector-size="${SECTOR_SIZE}" || true
env LOTUS_PATH="${ldt0222}" LOTUS_SEALER_PATH="${mdt0222}" ./lotus-sealer run --api 40001 &
sleep 10

env LOTUS_PATH="${ldt0222}" LOTUS_SEALER_PATH="${mdt0222}" LOTUS_DEALER_PATH="${ddt0222}" ./lotus-dealer init --sector-size="${SECTOR_SIZE}"

env LOTUS_PATH="${ldt0222}" LOTUS_SEALER_PATH="${mdt0222}" LOTUS_DEALER_PATH="${ddt0222}" ./lotus-dealer run --dealer-api 50001 &
