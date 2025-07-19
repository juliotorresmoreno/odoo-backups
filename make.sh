#!/bin/bash

mkdir -p bin
go build -o bin/odoo-backups main.go

docker build -t jliotorresmoreno/odoo-backups:v1.0.0 .

docker push jliotorresmoreno/odoo-backups:v1.0.0
