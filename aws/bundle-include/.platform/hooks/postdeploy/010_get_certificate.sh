#!/usr/bin/env bash
# TODO fix domain name
# TODO fix email
sudo certbot \
    -n \
    -d api.klemis-kitchen.ga \
    --nginx \
    --agree-tos \
    --email jazevedo620@gmail.com
