# yyabws

The this repository can be tested with the help of Docker and Docker compose.

The downstream [client side] rate limiter is implemented by the `rlmd` package [`rate limiting downstream`]. This packages depends on a Redis database

The server uses a redis database to keep track of user secrets a and the corresponding rate limiting state. 

The upstream [API side] rate limiting manager is implemented by the `rlmu` package [`rate limiting upstream`]

## Building 

### Regenerate the users secrets file if wanted. 
This file is copied to `WORKDIR` of the server in the server docker image and is used to initialize the Redis based rate limiter. 

```bash
./scripts/gen-secrets.sh
```
### Edit [.env](.env)
Update the `ATLAS_SECRET` value in [.env](.env) file.

### Build and run
Building:
```
docker-compose build
```
Running:

```bash
docker-compose up
```
Checking:
```bash
docker-compose ps
```
From the local host: 
Chose a secret from the `secrets.json`
```
tail secrets.json
```
Example:
```
 tail secrets.json 
"196b9a58a213463f897130c8ef8496b9",
"96481fed66b445b5977982a87e45147c",
"434f684826074850be9ac296c1d82705",
"5c078c9a804940bc8f6d88bba5170b9a",
"cd7d63721c944297bb420f357ca86048",
"8ca1e2275eef4f299c0c9b0b5dc9994c",
"b8603e8f5f0c44b995fba3566a35fe9e",
"f0f0b875ad024d89b88d39d667a550cd",
"df01616e8fd345349aef5452c6abdeff"
]

```
Use the selected secret to send a request to Atlas throng the server container

```
curl -i --location --request GET 'http://localhost:8081/teams/live?secret=df01616e8fd345349aef5452c6abdeff&take=1'
```
Enter the server container and run a little stress test:
```
docker-compose exec -it server /bin/bash
root@bd72125b06a4:/usr/src/app#

```
Check the number of secrets:
```
root@bd72125b06a4:~# stress secrets 
Number of secrets:  26
root@bd72125b06a4:~# 

```
List the secrets 

```
root@bd72125b06a4:~# stress secrets -list
Number of secrets:  26
d038df5d7edf47688bd60699ffd0a685
3231db906a3e4f52a0025b0c609ae654
edc71a92f81f40d9829df108e5bf2a69
a23da1ed421149caac7d90a372457e13
612f87bd5aba441081cb82150307ec6a
145775c687b44b5fb2f178dd9991eb25
6e1e8bcf6eb94e71912dd825ea1aa317
806f40a64d744fb2afd555a248a3a851
66bcf1d5078841ca867ae19642589944
6295a2cfac8a4684a3ad35e023b9ef9d
81beb360454c4883a263db6e75a8576a
d758d6aced564ce0b53d85a03e10ff76
6a52f684463040e8a2a5e48df57cbb65
c6f27e0d250243b2a7c73e1264914cf5
91857f14ff8c4edb8b0b2a2ebf15fd73
d56024f283254101888f2612e13f123c
909d5fcd83dc448395b2e729ee522d0b
196b9a58a213463f897130c8ef8496b9
96481fed66b445b5977982a87e45147c
434f684826074850be9ac296c1d82705
5c078c9a804940bc8f6d88bba5170b9a
cd7d63721c944297bb420f357ca86048
8ca1e2275eef4f299c0c9b0b5dc9994c
b8603e8f5f0c44b995fba3566a35fe9e
f0f0b875ad024d89b88d39d667a550cd
df01616e8fd345349aef5452c6abdeff
root@bd72125b06a4:~# 

```

```
stress run -endpoint "http://localhost:80/players/live" -pause 1ms --clients 1 -rounds 10 -skip 20 -take 1
```
Example:
```
root@bf8e17d6f95b:/usr/src/app# stress run -endpoint "http://localhost:80/players/live" -pause 1ms --clients 1 -rounds 10 -skip 20 -take 1
2024-09-10T07:57:17.420Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "4", "X-RateLimit-Reset": "926", "Retry-After": "0"}
d038df5d7edf47688bd60699ffd0a685 200 OK
2024-09-10T07:57:17.509Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "3", "X-RateLimit-Reset": "836", "Retry-After": "0"}
d038df5d7edf47688bd60699ffd0a685 200 OK
2024-09-10T07:57:17.580Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "2", "X-RateLimit-Reset": "766", "Retry-After": "0"}
d038df5d7edf47688bd60699ffd0a685 200 OK
2024-09-10T07:57:17.645Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "1", "X-RateLimit-Reset": "700", "Retry-After": "0"}
d038df5d7edf47688bd60699ffd0a685 200 OK
2024-09-10T07:57:17.712Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 200 OK
2024-09-10T07:57:17.715Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 429 Too Many Requests
2024-09-10T07:57:17.717Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 429 Too Many Requests
2024-09-10T07:57:17.719Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 429 Too Many Requests
2024-09-10T07:57:17.722Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 429 Too Many Requests
2024-09-10T07:57:17.724Z	DEBUG	stress/stress.go:33	rate limits headers	{"X-RateLimit-Limit": "5", "X-RateLimit-Burst": "5", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1000", "Retry-After": "1000"}
d038df5d7edf47688bd60699ffd0a685 429 Too Many Requests
root@bf8e17d6f95b:/usr/src/app# 
```
This `-endpoint` can be one of:
- "http://localhost:80/players/live"
- "http://localhost:80/teams/live"
- "http://localhost:80/series/live"
The `-pause` parameter specifies the periodicity of the requests for each client, is a time.Duration.
The `-clients` parmameter specifies the number of clients to simulate: should not be 0 and should not be larger than the number of secrets.

There an help flag:
```
root@bf8e17d6f95b:/usr/src/app# stress run -h
Usage of run:
  -clients int
    	number of simulated clients (default 5)
  -endpoint string
    	endpoint (default "http://localhost:80/teams/live")
  -pause duration
    	request period (default 1s)
  -print
    	print the responses
  -rounds int
    	number of rounds (default 3)
  -secrets string
    	users secrets file (default "/usr/src/app/secrets.json")
  -skip int
    	the number of records to skip
  -take int
    	the number of records to take (default 1)
root@bf8e17d6f95b:/usr/src/app# 

```



