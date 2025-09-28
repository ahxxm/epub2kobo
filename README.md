# send2ereader

A self hostable service for sending EPUB files to a Kobo e-reader through the built-in browser.

## How To Run

### On Your Host OS

1. Have Node.js 16 or 20 installed
2. Install this service's dependencies by running `$ npm install`
3. (Optional) Install [Kepubify](https://github.com/pgaskin/kepubify) for converting EPUB files to Kobo's enhanced format. Have the kepubify executable in your PATH.
4. Start this service by running: `$ npm start` and access it on HTTP port 3001

### Containerized
1. You need [Docker](https://www.docker.com/) and [docker-compose](https://docs.docker.com/compose/) installed
2. Clone this repo (you need Dockerfile, docker-compose.yaml and package.json in the same directory)
```
git clone https://github.com/daniel-j/send2ereader.git
```
3. Build the image
```
docker compose build
```
4. run container (-d to keep running in the background)
```
docker compose up -d
```
5. Access the service on HTTP, default port 3001 (http://localhost:3001)
