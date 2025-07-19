FROM ubuntu:24.04

LABEL maintainer="juliotorres"

RUN mkdir -p /app
WORKDIR /app

COPY bin/odoo-backups .

RUN chmod +x /app/odoo-backups

USER ubuntu

EXPOSE 3050

ENTRYPOINT [ "/app/odoo-backups" ]
