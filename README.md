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
[andre@cube yyabws]$ make clean
yamba-container
Untagged: yamba-image:latest
Deleted: sha256:7b68e6c21ed073f9d6f584648de53dc0ec3b9793421963383dcc36a99a5a9d20
[andre@cube yyabws]$ make 
[+] Building 5.1s (11/11) FINISHED                                                                                                                                                                  docker:default
 => [internal] load build definition from Dockerfile                                                                                                                                                          0.0s
 => => transferring dockerfile: 214B                                                                                                                                                                          0.0s
 => [internal] load metadata for docker.io/library/golang:1.23                                                                                                                                                0.8s
 => [internal] load .dockerignore                                                                                                                                                                             0.0s
 => => transferring context: 2B                                                                                                                                                                               0.0s
 => [1/6] FROM docker.io/library/golang:1.23@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd                                                                                          0.0s
 => [internal] load build context                                                                                                                                                                             0.0s
 => => transferring context: 21.64kB                                                                                                                                                                          0.0s
 => CACHED [2/6] WORKDIR /usr/src/app                                                                                                                                                                         0.0s
 => CACHED [3/6] COPY go.mod ./                                                                                                                                                                               0.0s
 => CACHED [4/6] RUN go mod download && go mod verify                                                                                                                                                         0.0s
 => [5/6] COPY . .                                                                                                                                                                                            0.1s
 => [6/6] RUN go build -v -o /usr/local/bin/atlas-proxy ./...                                                                                                                                                 3.6s
 => exporting to image                                                                                                                                                                                        0.6s 
 => => exporting layers                                                                                                                                                                                       0.6s 
 => => writing image sha256:648f5cb09c04d2609e10f63e81bf2ace0782ab2519b1c56000521ffa553f2bab                                                                                                                  0.0s 
 => => naming to docker.io/library/yamba-image                                                                                                                                                                0.0s 
[andre@cube yyabws]$ make run                                                                                                                                                                                      
52421ea258a5fc0d353cf76ac649579647785f7ceda7a7f43159f7cbf5cb48c7                                                                                                                                                   
[andre@cube yyabws]$ curl --request GET 'http://localhost:8080/series/live?take=1&skip=3'
[{"id":10,"title":"Round 1","start":"2014-06-25T09:00:00Z","end":"2014-06-25T10:00:00Z","postponed_from":null,"deleted_at":null,"lifecycle":"over","tier":1,"best_of":1,"chain":[],"streamed":true,"bracket_position":null,"participants":[{"seed":1,"score":0,"forfeit":false,"roster":{"id":1234},"winner":false,"stats":null},{"seed":2,"score":1,"forfeit":false,"roster":{"id":1233},"winner":true,"stats":null}],"tournament":{"id":3},"substage":{"id":4064},"game":{"id":5},"matches":[{"id":4}],"casters":[],"broadcasters":[],"has_incident_report":false,"coverage":{"data":{"live":{"api":{"expectation":"unsupported","fact":"unsupported"},"cv":{"expectation":"unsupported","fact":"unsupported"}},"realtime":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}},"postgame":{"api":{"expectation":"unsupported","fact":"unsupported"},"server":{"expectation":"unsupported","fact":"unsupported"}}}},"format":{"best_of":1},"game_version":{"release":{"uuid":"33dfd0b4-d36a-4d95-9a94-55d66818c848","description":"CS:GO","release_date":"2012-08-21"}},"resource_version":1,"created_at":"2015-09-14T07:20:09Z","updated_at":"2023-10-23T14:58:59Z"}]
[andre@cube yyabws]$ 

```