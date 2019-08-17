#! /bin/bash

set -x

while true; do
  date
  curl -X POST https://dht-test-2-svuagpbosa-uc.a.run.app/test
  curl -X POST https://dht-test-2-svuagpbosa-ue.a.run.app/test
  curl -X POST https://dht-test-2-svuagpbosa-ew.a.run.app/test
  curl -X POST https://dht-test-2-svuagpbosa-an.a.run.app/test
done
