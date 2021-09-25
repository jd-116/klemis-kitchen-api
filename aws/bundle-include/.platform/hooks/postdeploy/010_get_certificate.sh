#!/usr/bin/env bash

# This file was created based on this tutorial:
# https://medium.com/edataconsulting/how-to-get-a-ssl-certificate-running-in-aws-elastic-beanstalk-using-certbot-6daa9baa3997

# Any variables in this file get templated with envsubst
# during bundle creation.
# See aws/generate-bundle.sh

sudo certbot run \
    --non-interactive \
    --domains "${tls_domains}" \
    --nginx \
    --agree-tos \
    --email "${tls_email}" ${tls_staging}
