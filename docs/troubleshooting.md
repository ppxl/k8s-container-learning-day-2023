# Troubleshooting testclusters-go

## Remove remaining testclusters-go instances

Usually the cluster removes itself once the test exits in a normal way. But killing test processes might leave the started containers up. 

Remove remaining containers like this:

```bash
docker ps -f name=k3d-hello-world --format "{{.Names}}" | xargs docker rm -f
```

Remove remaining container networks like this:

```bash
docker network list -f name=k3d-hello --format "{{.Name}}" | xargs docker network rm
```