# DEAGER CLI

## Build Linux

```
$ docker-compose up
Starting deager_build
Attaching to deager_build
deager_build | + apt-get update -qq
deager_build | + apt-get install -qq -y make git golang
*snip*
deager_build | 187 added, 0 removed; done.
deager_build | Running hooks in /etc/ca-certificates/update.d...
deager_build | done.
deager_build | + cd /usr/local/src/github.con/apeltzer/deager/
deager_build | + go get -d
deager_build | + go build -o amd64/deager
deager_build | + echo 'La Fin!'
deager_build | La Fin!
deager_build exited with code 0
Gracefully stopping... (press Ctrl+C again to force)
```

