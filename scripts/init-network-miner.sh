#!/usr/bin/env bash

set -xeo

SECTOR_SIZE=2KiB

sdt0111="${HOME}/.sector-0" 
sdt0222="${HOME}/.sector-1" 
sdt0333="${HOME}/.sector-2" 

ldt0111="${HOME}/.lotus-0" 
ldt0222="${HOME}/.lotus-1" 
ldt0333="${HOME}/.lotus-2" 

mdt0111="${HOME}/.lotusminer-t01000" 
mdt0222="${HOME}/.lotusminer-t01001" 
mdt0333="${HOME}/.lotusminer-t01002" 

rm -rf $mdt0111
rm -rf $mdt0222
rm -rf $mdt0333

env LOTUS_PATH="${ldt0111}" LOTUS_MINER_PATH="${mdt0111}" ./lotus-miner init --genesis-miner --actor=t01000 --pre-sealed-sectors="${sdt0111}" --pre-sealed-metadata="${sdt0111}/pre-seal-t01000.json" --nosync=true --sector-size="${SECTOR_SIZE}" || true
env LOTUS_PATH="${ldt0111}" LOTUS_MINER_PATH="${mdt0111}" ./lotus-miner run --miner-api 40000 --nosync &