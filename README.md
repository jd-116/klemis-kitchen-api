# Klemis Kitchen API

> Backend API for the inventory app and web dashboard for the Klemis Kitchen,
> a food kitchen for food-insecure students at Georgia Tech.
>
> Created as a part of the multi-part GT CS Junior Design course in the Spring and Fall of 2020.

## 📃 Release Notes

**Current version**: v0.2.0

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

### v0.2.0 - AWS deployment functionality (2020-11-23)

#### Added

-   Support for deploying the application and related infrastructure on AWS. See [./aws](./aws) for more information
-   `--log-format` CLI parameter to the main binary, which can be either `console` or `json`
-   (internal) Switched to structured logging based on [zerolog](https://github.com/rs/zerolog)

### v0.1.1 - Locations Patch (2020-11-23)

#### Added

-   Support added for multiple locations, with Transact API scraping now properly detecting and separating items based on locations
-   (internal) Switched to a CSV-report-based scraping tool instead of directly requesting the inventory table. While more roundabout, this allows us to correctly obtain the location information reliably

### v0.1.0 - Initial Release (2020-11-22)

#### Added

-   Basic health check endpoint from `/v1/health` that always returns a `204 no content route`
-   [Working authentication API routes](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) that integrate with the Georgia Tech Single Sign-On service using the CAS protocol
-   [API for creating, deleting, updating, and viewing announcements](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) that are displayed in the Klemis Kitchen mobile app
-   [API for creating, deleting, updating, and viewing location metadata](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) that is used to create pins on the interactive map
-   [API for uploading images to Amazon S3](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) to be used for product thumbnails and nutritional information
-   [API for creating, deleting, updating, and viewing memberships](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) to the Klemis Kitchen, including whether they have access to the admin dashboard or not
-   [API for creating, deleting, updating, and viewing products](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design#locations) that are stocked at Klemis Kitchen locations, including fuzzy search functionality
-   (internal) Scraping logic that maintains an active session with the Transact Campus website and uses it to periodically fetch products from the existing point-of-sale system that Klemis Kitchen uses to manage inventory and "sales"
-   (internal) Logic to maintain an active session with a MongoDB database to persistently store all data that doesn't reside in the Transact Campus system or on Amazon S3

More details about the API routes can be found at the wiki page: [API Design](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design).

#### Known Issues

-   The current Docker-based hosting pattern occasionally sees transient connection issues in our staging environment that we're unsure of the cause. This may be related to the networking setup of the staging environment.
-   The API currently only supports a single location

---

## 🚀 Running (Install Guide)

> Note that this guide is intended for an audience with a moderate level of technical literacy, especially when it comes to using Linux from a terminal and administering a server.

### Prerequisites

-   Linux-based machine
-   Docker engine - https://docs.docker.com/engine/install/#server

Once the prerequisites are installed, the API will need to be built. First, make sure to download the Git repository by running `git clone` as normal or download a ZIP file from Github.com. Then, run the following command in a terminal (from the project root):

```sh
docker build ./api -t klemis-kitchen-api
```

This runs our custom build script and installs all internal dependencies so the only software you need is Docker.

Then, before you can run the application, you need to have the environment variables configured. An example of the values that need to be set is in `.env.example`. To use this, **make a copy called .env** in the same folder and change the values to include the tokens/secrets/values as needed. A breakdown of the values needed can be found below in the _"⚙ Configuration"_ section.

Finally, to run the API server, run:

```sh
docker run -d \
    --name klemis-kitchen-api \
    --mount type=bind,source="$(pwd)"/.env,target=/etc/klemis-kitchen-api.env,readonly \
    --env PORT=8080 \
    --publish 8080:8080 \
    klemis-kitchen-api:latest \
    --env /etc/klemis-kitchen-api.env
```

which will start the API server in the background, listening for and responding to HTTP traffic at `SERVER_PORT` (configured in `.env`). Note that both the mobile app and admin dashboard are packaged and pre-configured to attempt to connect to the API at `https://backend.klemis-kitchen.com`, so DNS will need to be configured to point to the host server.

More details about the API routes that the server provides can be found at the wiki page: [API Design](https://github.com/jd-116/klemis-kitchen-api/wiki/API-Design).

### Securing with HTTPS

While the information served by the Klemis Kitchen API isn't necessarily sensitive, it is considered good practice to serve traffic using HTTPS. Additionally, because of the way CAS (the protocol used to connect to the Georgia Tech Single Sign-On service) works **authentication may not work correctly if the API isn't behind HTTPS**.

To fix this, some external load balancer or server needs to sit in front of the API server and perform **SSL termination** since the API server itself is unable to serve HTTPS traffic.

A Docker-based solution is available by using [`nginx-proxy`](https://github.com/nginx-proxy/nginx-proxy), which contains a Docker container that performs SSL termination in addition to automatically provisioning valid SSL certificates using LetsEncrypt. The commands to run it and the API server as a handful of Docker containers are as follows (note the additional parameters on the API server container; these are needed):

```sh
docker run --detach \
    --name nginx-proxy \
    # Change the first number here (8008) to the external port for HTTP
    --publish 8008:80 \
    # Change the first number here (8043) to the external port for HTTPS
    --publish 8043:443 \
    --volume /srv/http/klemis-kitchen/certs:/etc/nginx/certs \
    --volume /srv/http/klemis-kitchen/vhost:/etc/nginx/vhost.d \
    --volume /srv/http/klemis-kitchen/html:/usr/share/nginx/html \
    --volume /var/run/docker.sock:/tmp/docker.sock:ro \
    jwilder/nginx-proxy

docker run --detach \
    --name nginx-proxy-letsencrypt \
    --volumes-from nginx-proxy \
    --volume /var/run/docker.sock:/var/run/docker.sock:ro \
    # Change the email here which is used to manage the certificates
    --env "DEFAULT_EMAIL=klemis.kitchen.management@gmail.com" \
    jrcs/letsencrypt-nginx-proxy-companion

docker run -d \
    --name klemis-kitchen-api \
    --mount type=bind,source="$(pwd)"/.env,target=/etc/klemis-kitchen-api.env,readonly \
    --env PORT=8080 \
    --env "VIRTUAL_PORT=8080" \
    # Change both environment parameters here
    # if the server is hosted at a different URL
    --env "VIRTUAL_HOST=backend.klemis-kitchen.com" \
    --env "LETSENCRYPT_HOST=backend.klemis-kitchen.com" \
    klemis-kitchen-api:latest \
    --env /etc/klemis-kitchen-api.env \
    --log-format json
```

### Troubleshooting

-   "docker: command not found": make sure you have followed the instructions to [install the Docker engine](https://docs.docker.com/engine/install/#server)
-   File not found (at any point in the install process): make sure the commands are all run in the project root, and the Docker engine needs to be installed on the machine.
-   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?": The Docker engine is likely not running on the machine. If running on Linux machines with Systemd, the Docker engine can be started by running:

    ```
    sudo systemctl start docker
    ```

-   "502 Bad Gateway": this page can appear if the API server is still initializing when using the `nginx-proxy` setup. Wait a couple minutes and try again.
-   Other general errors: make sure that you have filled in all required fields in the `.env` file and that the path to the `.env` file is correct when passed into `docker run`
-   Authentication doesn't work: make sure that you're running the server with HTTPS enabled so that CAS is able to properly work and release all needed information.

### FAQ

**Q: What type of machine does this run on?<br>**
A: The API runs on a Linux machine

**Q: What prerequisites need to be installed?<br>**
A: Docker needs to be installed (to build and run as a Docker image). Otherwise, the application can be built using the Go compiler, and it might have various shared library dependencies at runtime.

**Q: How does the API work?<br>**
A: The API, written in Golang, scrapes data from Klemis kitchen's PoS system through Transact (the online dashboard). Data is stored in MongoDB and S3 (more resource intensive data). Look at our Detailed Design Document for more information.

**Q: What values need to go in the .env file?<br>**
A: Read the configuration section for more information on environment variables

**Q: Why does the server need HTTPS?<br>**
A: The Georgia Tech Single Sign-On service requires third-party applications to use HTTPS when releasing information about users, such as their first/last name and GT username. Because this is used in the authentication pathway, the server needs to be accessible via HTTPS.

**Q: How do I set up HTTPS?<br>**
A: The way HTTPS is set up doesn't matter for the application; all that matters is that it exists and can be used by clients. The guide above provides a method that uses Docker for convenience, but any method that serves HTTPS connections with a valid SSL/TLS certificate can work. For example, [this guide](https://www.nginx.com/blog/using-free-ssltls-certificates-from-lets-encrypt-with-nginx/) goes over the process of using LetsEncrypt certificates with Nginx on bare metal.

## ⚙ Configuration

This section describes each environment variable that the API supports. If running on AWS, see `./aws/terraform.tfvars.example` for the configuration file and reference information.

#### API host parameters

```
API_SERVER_DOMAIN=
```

Provide the server domain the API exposes for the application to communicate with

#### Authentication parameters

```sh
# Whether the flow continuation cookies used between redirects should be secure (HTTPS-only)
AUTH_SECURE_CONTINUATION=0
# List of prefixes to match authentication redirect URIs against (should include the admin dashboard).
# If empty, then all URIs are allowed.
AUTH_REDIRECT_URI_PREFIXES=
# The (base64-encoded) encryption secret used for signing JWTs (should be between 128 and 512 bits)
AUTH_JWT_SECRET=secret
# The number of hours after which to expire JWTs (and require re-authentication). Empty disables expiration
AUTH_JWT_TOKEN_EXPIRES_AFTER=
# Whether to disable authentication completely. Do not run this in production!
AUTH_BYPASS=1
```

#### MongoDB connection credentials

```sh
# The username for a MongoDB Atlas account that can access the API's database instance
MONGO_DB_USERNAME=
# The password for a MongoDB Atlas account that can access the API's database instance
MONGO_DB_PASSWORD=
# The name of the MongoDB Atlas cluster that the app is running on
MONGO_DB_CLUSTER_NAME=
# The name of the MongoDB database (collection of collections) that all of the API's collections should reside in
MONGO_DB_DATABASE_NAME=
```

#### Transact API connection credentials/parameters

```sh
# The base URL of the Transact API to retrieve inventory data from
TRANSACT_BASE_URL="https://qpc.transactcampus.com"
# The 'tenant' in Transact to that the API should authenticate against and download inventory for
TRANSACT_TENANT=gatech
# The username for a Transact account that can access and execute favorite reports to obtain inventory data
TRANSACT_USERNAME=
# The password for a Transact account that can access and execute favorite reports to obtain inventory data
TRANSACT_PASSWORD=
# The period to wait between fetches of the current inventory.
# This affects data liveness served by the API as well as the load induced on Transact
TRANSACT_FETCH_PERIOD=10m
# The period to wait between 'reloading' the Transact session (simulating logging out and back in again)
TRANSACT_RELOAD_SESSION_PERIOD=30m
# The name of the favorite report created in Transact
# that should be based on 'Item List with Inventory Details' and output CSV
TRANSACT_CSV_FAVORITE_REPORT_NAME="Klemis Inventory CSV"
# The period to wait between seeing if a newly-requested report is ready to download
TRANSACT_REPORT_POLL_PERIOD=10s
# The period to wait before giving up on a requested report after which it errors
TRANSACT_REPORT_POLL_TIMEOUT=5m
# The 0-based offset for the cell that the product's name exists in,
# relative to the cell that indicates the profit center
TRANSACT_CSV_REPORT_ID_COLUMN_OFFSET=9
# The 0-based offset for the cell that the product's ID exists in,
# relative to the cell that indicates the profit center
TRANSACT_CSV_REPORT_NAME_COLUMN_OFFSET=10
# The 0-based offset for the cell that the product's current quantity exists in,
# relative to the cell that indicates the profit center
TRANSACT_CSV_REPORT_QTY_COLUMN_OFFSET=13
# The prefix that exists in each cell that also contains the profit center.
# For example, 'Profit Center -' matches cells with the contents:
# - 'Profit Center - Pantry A'
#    (which turns into the profit center name 'Pantry A')
# - 'Profit Centry - Location 002'
#    (which turns into the profit center name 'Location 002')
TRANSACT_PROFIT_CENTER_PREFIX="Profit Center -"
# The expected '__type' field of the report that the scraper searches for.
# This is an internal value in the Transact API
TRANSACT_CSV_REPORT_TYPE="qpsview_reports_schedules:#QPWebOffice.Web"
```

#### CAS login arguments

```sh
# The base URL (including the trailing '/cas/')
# for the CAS (single-sign-on) server that is used to authenticate users.
# The API uses CAS protocol version 2 to implement communication with the SSO provider:
# https://apereo.github.io/cas/5.1.x/protocol/CAS-Protocol-V2-Specification.html
CAS_SERVER_URL="https://login.gatech.edu/cas/"
```

#### Upload credentials/parameters

```sh
# The max size of files that can be uploaded using the API to S3
UPLOAD_MAX_SIZE=4GB
# The AWS region that the bucket should exist in
UPLOAD_AWS_REGION="us-east-1"
# The AWS access key ID to use when uploading files to S3
UPLOAD_AWS_ACCESS_KEY_ID=
# The AWS secret access key to use when uploading files to S3
UPLOAD_AWS_SECRET_ACCESS_KEY=
# The size of chunks to use when uploading files to S3
UPLOAD_PART_SIZE=6MB
# The name of the S3 bucket to upload files to
UPLOAD_S3_BUCKET=klemis-product-images
```

### 🧪 Testing with health check route

To test to make sure the API is running, open up a new terminal and run (make sure to change `8080` to whatever port the API server is accepting connections on):

```
curl -v localhost:8080/v1/health
```

The response should include something like this if it worked:

```
* Connected to localhost (::1) port 8080 (#0)
> GET /v1/health HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.71.1
> Accept: */*
...
< HTTP/1.1 204 No Content
```
