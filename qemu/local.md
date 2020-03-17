# Running Local Environment

## Run local mock
- You need a mongodb: `docker run --net host --name mockmongo -d mongo:latest`
- Run mock locally: `./tools/bcdb_mock/bcdb_mock`

## Register a user and a farm on the mock
```
tfuser --bcdb http://127.0.0.1:8080 id --output /tmp/zos/local.seed --name debug --email debug@localhost --description local
```

## Register a new farm
```
tffarmer --bcdb http://127.0.0.1:8080 farm register debug --tid 1
```

## Send a test provision
```
tfuser generate container --flist nonexisting > /tmp/zos/newtest.json
tfuser --bcdb http://127.0.0.1:8080 provision --schema /tmp/zos/newtest.json --node nonexisting \
    --seed /tmp/zos/local.seed --id 1 --duration 1m
```

## QEMU
Start a vm and point the bcdb to your local mock
```
bash vm.sh -b lan -n node1 -c "runmode=dev farmer_id=1 bcdb=http://10.241.0.106:8080"
```
