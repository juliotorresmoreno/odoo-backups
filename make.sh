#!/bin/bash

docker build -t jliotorresmoreno/odoo-backups:v1.0.0 .

docker push jliotorresmoreno/odoo-backups:v1.0.0
