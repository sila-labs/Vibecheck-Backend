# VibeCheck Location API

#### Run via Terminal (LOCALLY)

``` 
pip install -r requirements.txt
```
#### Run Redis Cache via Docker (REQUIRED) br
``` BASH (Windows Does Not Use 'sudo' Command)
sudo docker run --name viberedis -d -p6465:6379 redis
```

```
uvicorn app:app --reload --port 6464
```

```
python3 -m uvicorn app:app --reload --port 6464
```


#### Run via Docker

```
docker buildx build --platform=linux/amd64 -t vibcheck-location-api .
```

```
docker run --name vla -p 6464:64646 vibcheck-location-api
```
