#!/bin/bash
# wait-for-postgres.sh

set -e

HOST="$1"
shift
CMD="$@"

until nc -z -v -w30 $HOST 5432
do
  echo "Waiting for Postgres connection..."
  # wait for 5 seconds before check again
  sleep 5
done

>&2 echo "Postgres is up - executing command"
exec $CMD