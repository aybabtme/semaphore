#!/usr/bin/env bash

lockname=$(uuidgen)
semaphore create --name ${lockname} --size 2

function fetch_url() {
    local url=${1}
    semaphore acquire --name ${lockname}
    echo "fetching URL ${1}"
    sleep 1
    semaphore release --name ${lockname}
}

for ((i=0; i<=10; i++)); do
    fetch_url "http://url.number.${i}" &
done

wait $(jobs -p)