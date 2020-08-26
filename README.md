# Klemis Kitchen API

> Backend API for the inventory app for the Klemis Kitchen,
> a food kitchen for food-insecure students at Georgia Tech.
>
> Created as a part of the multi-part GT CS Junior Design course.

## 🚀 Running Locally

To run the API locally, you need to have Docker and Docker Compose installed.
On desktop systems like
[Docker Desktop](https://www.docker.com/products/docker-desktop) for Mac or Windows,
Docker Compose is included as part of those desktop installs.
On Linux, the [Docker Engine](https://docs.docker.com/engine/install/#server)
needs to be installed separately before Docker Compose can be installed
by following the instructions on their [Install page](https://docs.docker.com/compose/install/).

Once installed, you'll need to configure the environment variables.
To do so, copy `.env.example` in the repository root to `.env` and fill in the missing fields.
You'll need an active MongoDB connection and credentials to run the API server.

Finally, start up the application container by running:

```
docker-compose up
```

The API should then be accessible at `http://localhost:8080`.

### 🧪 Testing with health check route

To test to make sure the API is running, open up a new terminal and run:

```
curl -v localhost:8080/api/v1/health
```

The response should include something like this if it worked:

```
* Connected to localhost (::1) port 8080 (#0)
> GET /api/v1/health HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.71.1
> Accept: */*
...
< HTTP/1.1 204 No Content
```
