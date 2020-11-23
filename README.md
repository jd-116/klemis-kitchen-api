# Klemis Kitchen API

> Backend API for the inventory app and web dashboard for the Klemis Kitchen,
> a food kitchen for food-insecure students at Georgia Tech.
>
> Created as a part of the multi-part GT CS Junior Design course in the Spring and Fall of 2020.

## 📃 Release Notes

**Current version**: v0.1.0

### Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

#### v0.1.0 - Initial Release (2020-11-22)

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
    --env-file ./.env \
    --env SERVER_PORT=8080 \
    --publish 8080:8080 \
    klemis-kitchen-api:latest
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
    --env-file ~/dev/klemis-kitchen-api/.env \
    --env SERVER_PORT=8080 \
    --env "VIRTUAL_PORT=8080" \
    # Change both environment parameters here
    # if the server is hosted at a different URL
    --env "VIRTUAL_HOST=backend.klemis-kitchen.com" \
    --env "LETSENCRYPT_HOST=backend.klemis-kitchen.com" \
    klemis-kitchen-api:latest
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
A: Docker needs to be installed

**Q: How does the API work?<br>**
A: The API, written in Golang, scrapes data from Klemis kitchen's PoS system through Transact (the online dashboard). Data is stored in MongoDB and S3 (more resource intensive data). Look at our Detailed Design Document for more information.

**Q: What values need to go in the .env file?<br>**
A: Read the configuration section for more information on environment variables

**Q: Why does the server need HTTPS?<br>**
A: The Georgia Tech Single Sign-On service requires third-party applications to use HTTPS when releasing information about users, such as their first/last name and GT username. Because this is used in the authentication pathway, the server needs to be accessible via HTTPS.

**Q: How do I set up HTTPS?<br>**
A: The way HTTPS is set up doesn't matter for the application; all that matters is that it exists and can be used by clients. The guide above provides a method that uses Docker for convenience, but any method that serves HTTPS connections with a valid SSL/TLS certificate can work. For example, [this guide](https://www.nginx.com/blog/using-free-ssltls-certificates-from-lets-encrypt-with-nginx/) goes over the process of using LetsEncrypt certificates with Nginx on bare metal.

## ⚙ Configuration

#### API host parameters

```
API_SERVER_DOMAIN=
```

Provide the server domain the API exposes for the application to communicate with

#### Authentication parameters

```
AUTH_SECURE_CONTINUATION=0
AUTH_REDIRECT_URI_PREFIXES=
AUTH_JWT_SECRET=
AUTH_JWT_TOKEN_EXPIRES_AFTER=
AUTH_BYPASS=0
```

The required authentication credentials for users to utilize the application. Provide a base 64 encoded secret for the `AUTH_JWT_SECRET` field. The `AUTH_JWT_TOKEN_EXPIRES_AFTER` is the number of hours after a token is issued that it expires. If empty, the tokens default to token not expiring.

#### MongoDB connection credentials

```
MONGO_DB_HOST=
MONGO_DB_PWD=
MONGO_DB_CLUSTER=
MONGO_DB_NAME=
```

MongoDB credentials are derived from Mongodb Atlas. The `MONGO_DB_HOST` and `MONGO_DB_PWD` are derived after creating an account with MongoDB Atlas, and `MONGO_DB_CLUSTER` and `MONGO_DB_NAME` are for the cluster on Atlas used to store data.

#### Transact API connection credentials/parameters

```
TRANSACT_BASE_URL=https://qpc.transactcampus.com
TRANSACT_TENANT=gatech
TRANSACT_USERNAME=
TRANSACT_PASSWORD=
TRANSACT_FETCH_PERIOD=10m
TRANSACT_RELOAD_SESSION_PERIOD=30m
TRANSACT_PRODUCT_CLASS_NAME=Klemis Pantry
```

Transact credentials are defined for Georgia Tech food services credentials. The `TRANSACT_USERNAME` and `TRANSACT_PASSWORD` are provided by STAR services. The `TRANSACT_FETCH_PERIOD` updates the quantity in the database from the live Transact stock every fetch period. `TRANSACT_RELOAD_SESSION_PERIOD` reloads the connection to Transact with provided credentials every session period.

#### CAS login arguments

```
CAS_SERVER_URL=https://login.gatech.edu/cas/
```

`CAS_SERVER_URL` defines the CAS url used for authentication.

#### Upload credentials/parameters

```
UPLOAD_MAX_SIZE=4GB
UPLOAD_AWS_REGION=us-east-1
UPLOAD_AWS_ACCESS_KEY_ID=
UPLOAD_AWS_SECRET_ACCESS_KEY=
UPLOAD_PART_SIZE=6MB
UPLOAD_S3_BUCKET=klemis-product-images
```

The following parameters are essential to S3 data storage, where nutritional facts images and thumbnails are stored. The `UPLOAD_MAX_SIZE` and `UPLOAD_AWS_REGION` define the maximum size and regional configuration for the bucket. `UPLOAD_AWS_ACCESS_KEY_ID` and `UPLOAD_AWS_SECRET_ACCESS_KEY` are AWS credentials unique for each account.

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
