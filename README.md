# semaphore

A shell tool to create counting semaphores, acquire them and release them. This is useful if you want to e.g. run no more than N out of M commands in parallel.

## usage

Create a semaphore of size 10 for a job that will fetch URLs in parallel.

```bash
semaphore create --name fetch-many-urls --size 10
```

Then before launching each job, acquire the semaphore:

```bash
semaphore acquire --name fetch-many-urls
```

Do your job, and when you're done:

```bash
semaphore release --name fetch-many-urls
```

### example

```bash
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
```
