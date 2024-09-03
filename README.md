# yyabws

Docker is needed for testing.

The makefile reads environment variables from the [`.env`](.env) file.

## To build the docker image
Please use the make command:
This will build a docker image from the sources
```bash
make 
```
## To run the docker image:
Please use the make command:
```bash
make run
```

## To stop everything:
Please run the make command:
```bash
make clean
```
This stops the docker container, removes it and removes the created docker image.

## Example showing all the above described steps.
```
➜  yyabws git:(main) ✗ make
[+] Building 5.1s (11/11) FINISHED                                                                                                                                                                  docker:default
 => [internal] load build definition from Dockerfile                                                                                                                                                          0.0s
 => => transferring dockerfile: 214B                                                                                                                                                                          0.0s
 => [internal] load metadata for docker.io/library/golang:1.23                                                                                                                                                0.9s
 => [internal] load .dockerignore                                                                                                                                                                             0.0s
 => => transferring context: 2B                                                                                                                                                                               0.0s
 => [1/6] FROM docker.io/library/golang:1.23@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd                                                                                          0.0s
 => [internal] load build context                                                                                                                                                                             0.0s
 => => transferring context: 3.54kB                                                                                                                                                                           0.0s
 => CACHED [2/6] WORKDIR /usr/src/app                                                                                                                                                                         0.0s
 => CACHED [3/6] COPY go.mod ./                                                                                                                                                                               0.0s
 => CACHED [4/6] RUN go mod download && go mod verify                                                                                                                                                         0.0s
 => [5/6] COPY . .                                                                                                                                                                                            0.0s
 => [6/6] RUN go build -v -o /usr/local/bin/atlas-proxy ./...                                                                                                                                                 3.6s
 => exporting to image                                                                                                                                                                                        0.5s 
 => => exporting layers                                                                                                                                                                                       0.5s 
 => => writing image sha256:e9a9f1e7111d1007de76a6c68c611335bcef92ae1606757334b778f96a11ed30                                                                                                                  0.0s 
 => => naming to docker.io/library/yamba-image                                                                                                                                                                0.0s 
➜  yyabws git:(main) ✗ make run
7c5ce176fb7ef31a40a0f10fa74885275ab5f513786f015d8faf747051984ca2
➜  yyabws git:(main) ✗ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED         STATUS         PORTS                                   NAMES
7c5ce176fb7e   yamba-image            "atlas-proxy"            5 seconds ago   Up 5 seconds   0.0.0.0:8080->80/tcp, :::8080->80/tcp   yamba-container
d8e588c19a07   kindest/node:v1.30.0   "/usr/local/bin/entr…"   2 months ago    Up 31 hours    127.0.0.1:39009->6443/tcp               kind-control-plane
➜  yyabws git:(main) ✗ curl -i --location --request GET 'http://localhost:8080/series/live?take=2&skip=3'  
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 03 Sep 2024 13:52:09 GMT
Transfer-Encoding: chunked

[{"id":10,"title":"Round 1","start":"2014-06-25T09:00:00Z","end":"2014-06-25T10:00:00Z","postponed_from":null,"deleted_at":null,"lifecycle":"over","tier":1,"best_of":1,"chain":[],"streamed":true,"bracket_position":null,"participants":[{"seed":1,"score":0,"forfeit":false,"roster":{"id":1234},"winner":false,"stats":null},{"seed":2,"score":1,"forfeit":false,"roster":{"id":1233},"winner":true,"stats":null}],"tournament":{"id":3},"substage":{"id":4064},"game":{"id":5},"matches":[{"id":4}],"casters":[],"broadcasters":[],"has_incident_report":false,"coverage":{"data":{"live":{"api":{"expectation":"unsupported","fact":"unsupported"},"cv":{"expectation":"unsupported","fact":"unsupported"}},"realtime":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}},"postgame":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}}}},"format":{"best_of":1},"game_version":{"release":{"uuid":"33dfd0b4-d36a-4d95-9a94-55d66818c848","description":"CS:GO","release_date":"2012-08-21"}},"resource_version":1,"created_at":"2015-09-14T07:20:09Z","updated_at":"2023-10-23T14:58:59Z"},{"id":11,"title":"Round 1","start":"2014-06-26T18:00:00Z","end":"2014-06-26T19:00:00Z","postponed_from":null,"deleted_at":null,"lifecycle":"over","tier":1,"best_of":1,"chain":[],"streamed":true,"bracket_position":null,"participants":[{"seed":1,"score":1,"forfeit":false,"roster":{"id":1230},"winner":true,"stats":null},{"seed":2,"score":0,"forfeit":false,"roster":{"id":1235},"winner":false,"stats":null}],"tournament":{"id":3},"substage":{"id":4064},"game":{"id":5},"matches":[{"id":5}],"casters":[],"broadcasters":[],"has_incident_report":false,"coverage":{"data":{"live":{"api":{"expectation":"unsupported","fact":"unsupported"},"cv":{"expectation":"unsupported","fact":"unsupported"}},"realtime":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}},"postgame":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}}}},"format":{"best_of":1},"game_version":{"release":{"uuid":"33dfd0b4-d36a-4d95-9a94-55d66818c848","description":"CS:GO","release_date":"2012-08-21"}},"resource_version":1,"created_at":"2016-05-19T15:23:10Z","updated_at":"2023-10-23T14:58:59Z"}]%                                                                                                                                                ➜  yyabws git:(main) ✗ make clean
yamba-container
Untagged: yamba-image:latest
Deleted: sha256:e9a9f1e7111d1007de76a6c68c611335bcef92ae1606757334b778f96a11ed30
➜  yyabws git:(main) ✗ docker ps                                                                           
CONTAINER ID   IMAGE                  COMMAND                  CREATED        STATUS        PORTS                       NAMES
d8e588c19a07   kindest/node:v1.30.0   "/usr/local/bin/entr…"   2 months ago   Up 32 hours   127.0.0.1:39009->6443/tcp   kind-control-plane
➜  yyabws git:(main) ✗ 
```

