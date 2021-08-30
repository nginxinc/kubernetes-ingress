
#install app dos

## kind

### create cluster
```
kind create cluster
```

### create app dos image
```
make debian-image-napdos-plus
```

### find and save the tagged image name from the make output, i.e.
```
...
 -t nginx/nginx-ingress:1.12.0-SNAPSHOT-5c5f194 
...
```

### 