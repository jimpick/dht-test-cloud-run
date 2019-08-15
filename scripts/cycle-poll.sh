#! /bin/bash

set -x

while true; do
  date
  curl -X POST https://dht-test-svuagpbosa-uc.a.run.app/test
  curl -X POST https://dht-test-svuagpbosa-ue.a.run.app/test
  curl -X POST https://dht-test-svuagpbosa-ew.a.run.app/test
  curl -X POST https://dht-test-svuagpbosa-an.a.run.app/test
done
