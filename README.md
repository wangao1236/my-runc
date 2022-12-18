# my-runc

It is a personal implementation of a simple container runtime.

It references some features from https://github.com/xianlubird/mydocker, and improvements have also been made to some
methods, such as using OverlayFs instead of AuFs.

It implements containers' creation, deletion and list. It also implements a simple overlay network of containers.

## examples

### networks

#### create a network

```bash
$ ./bin/my-docker network create --subnet 192.168.60.1/24 test-bridge
```

### list networks

```bash
$ ./bin/my-docker network list

NAME          SUBNET            GATEWAY        DRIVER
test-bridge   192.168.60.0/24   192.168.60.1   bridge
```

### delete a network

```bash
$ ./bin/my-docker network delete test-bridge
```

#### containers

#### run a container

```bash
PATH=/bin:$PATH ./bin/my-docker run -d -name test1 -v /data/test-hostpath/from1:/to1 --mem 10000m -e key1=val1 -e key2=val2 --network test-bridge -p 11111:8089 -p 11112:8090 to
```

#### list containers

```bash
$ ./bin/my-docker ps

ID           NAME        PID         STATUS      COMMAND     CREATED                                   IP
1281457058   test1       11211       running     top         2022-12-18 13:06:57.241502751 +0800 CST   192.168.60.2/24
```

#### stop a container

```bash
$ ./bin/my-docker stop test1
```

### remove a container

```bash
$ ./bin/my-docker rm test1
```