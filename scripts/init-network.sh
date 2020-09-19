#!/usr/bin/env bash

set -xeo

NUM_SECTORS=2
SECTOR_SIZE=8MiB


sdt0111=$(mktemp -d)
sdt0222=$(mktemp -d)
sdt0333=$(mktemp -d)

sdt0111="${HOME}/.sector-0" 
sdt0222="${HOME}/.sector-1" 
sdt0333="${HOME}/.sector-2" 

staging=$(mktemp -d)
staging="${HOME}/.genesis" 

rm -rf $sdt0111
rm -rf $sdt0222
rm -rf $sdt0333
rm -rf $staging

mkdir $staging

make

./lotus-seed --sector-dir="${sdt0111}" pre-seal --miner-addr=t01000 --sector-offset=0 --sector-size=${SECTOR_SIZE} --num-sectors=${NUM_SECTORS} &
./lotus-seed --sector-dir="${sdt0222}" pre-seal --miner-addr=t01001 --sector-offset=0 --sector-size=${SECTOR_SIZE} --num-sectors=${NUM_SECTORS} &
./lotus-seed --sector-dir="${sdt0333}" pre-seal --miner-addr=t01002 --sector-offset=0 --sector-size=${SECTOR_SIZE} --num-sectors=${NUM_SECTORS} &

wait

./lotus-seed aggregate-manifests "${sdt0111}/pre-seal-t01000.json" "${sdt0222}/pre-seal-t01001.json" "${sdt0333}/pre-seal-t01002.json" > "${staging}/pre-seal-single.json"
./lotus-seed genesis new "${staging}/genesis.json"
./lotus-seed genesis add-miner "${staging}/genesis.json" "${staging}/pre-seal-single.json"


lotus_path=$(mktemp -d)

./lotus --repo="${lotus_path}" daemon --lotus-make-genesis="${staging}/devnet.car" --genesis-template="${staging}/genesis.json" --bootstrap=false &
lpid=$!

sleep 30

kill "$lpid"

wait

cp "${staging}/devnet.car" build/genesis/devnet.car

make

ldt0111=$(mktemp -d)
ldt0222=$(mktemp -d)
ldt0333=$(mktemp -d)

ldt0111="${HOME}/.lotus-0" 
ldt0222="${HOME}/.lotus-1" 
ldt0333="${HOME}/.lotus-2" 

rm -rf $ldt0111
rm -rf $ldt0222
rm -rf $ldt0333

sdlist=( "$sdt0111" "$sdt0222" "$sdt0333" )
ldlist=( "$ldt0111" "$ldt0222" "$ldt0333" )

pids=()
for (( i=0; i<${#ldlist[@]}; i++ )); do
  repo=${ldlist[$i]}
  ./lotus --repo="${repo}" daemon --api "3000$i" --bootstrap=false &
  pids+=($!)
done

sleep 10

for (( i=0; i<${#sdlist[@]}; i++ )); do
  preseal=${sdlist[$i]}
  fullpath=$(find ${preseal} -type f -iname 'pre-seal-*.json')
  filefull=$(basename ${fullpath})
  filename=${filefull%%.*}
  repo=${ldlist[$i]}
  ./lotus --repo="${repo}" wallet import --as-default "${preseal}/${filename}.key"
done

boot=$(./lotus --repo="${ldlist[0]}" net listen)

for (( i=1; i<${#ldlist[@]}; i++ )); do
  repo=${ldlist[$i]}
  ./lotus --repo="${repo}" net connect ${boot}
done

sleep 3

mdt0111=$(mktemp -d)
mdt0222=$(mktemp -d)
mdt0333=$(mktemp -d)

mdt0111="${HOME}/.lotusminer-t01000" 
mdt0222="${HOME}/.lotusminer-t01001" 
mdt0333="${HOME}/.lotusminer-t01002" 

rm -rf $mdt0111
rm -rf $mdt0222
rm -rf $mdt0333

env LOTUS_PATH="${ldt0111}" LOTUS_MINER_PATH="${mdt0111}" ./lotus-miner init --genesis-miner --actor=t01000 --pre-sealed-sectors="${sdt0111}" --pre-sealed-metadata="${sdt0111}/pre-seal-t01000.json" --nosync=true --sector-size="${SECTOR_SIZE}" || true
env LOTUS_PATH="${ldt0111}" LOTUS_MINER_PATH="${mdt0111}" ./lotus-miner run --nosync &
mpid=$!

env LOTUS_PATH="${ldt0222}" LOTUS_MINER_PATH="${mdt0222}" ./lotus-miner init                 --actor=t01001 --pre-sealed-sectors="${sdt0222}" --pre-sealed-metadata="${sdt0222}/pre-seal-t01001.json" --nosync=true --sector-size="${SECTOR_SIZE}" || true
env LOTUS_PATH="${ldt0333}" LOTUS_MINER_PATH="${mdt0333}" ./lotus-miner init                 --actor=t01002 --pre-sealed-sectors="${sdt0333}" --pre-sealed-metadata="${sdt0333}/pre-seal-t01002.json" --nosync=true --sector-size="${SECTOR_SIZE}" || true

kill $mpid
wait $mpid

for (( i=0; i<${#pids[@]}; i++ )); do
  kill ${pids[$i]}
done

wait

#rm -rf $mdt0111
#rm -rf $mdt0222
#rm -rf $mdt0333

#rm -rf $ldt0111
#rm -rf $ldt0222
#rm -rf $ldt0333

#rm -rf $sdt0111
#rm -rf $sdt0222
#rm -rf $sdt0333

#rm -rf $staging
