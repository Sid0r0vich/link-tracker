#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "generating users.csv..."
cd "$SCRIPT_DIR" || exit 1
seq 0 999 > users.csv
cd "$PROJECT_ROOT"

echo "cleaning up db..."
docker compose down -v

echo "run containers..."
docker compose up -d

echo "waiting for services to start..."
sleep 2

echo "add data to db..."
go run ./tests/load/db_prepare.go

cd "$SCRIPT_DIR" || exit 1

echo "cleaning up previous results..."
rm -rf results-report

echo "start jmeter test..."
jmeter -n -t test.jmx -l results.jtl -e -o results-report -j jmeter.log

echo "cleaning up..."
rm users.csv
rm results.jtl
rm jmeter.log

echo "stop containers..."
cd "$PROJECT_ROOT"
docker compose down -v