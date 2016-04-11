# DEAGER CLI
## Installation
This is the docker helper client for the EAGER Pipeline. Now we added precompiled binaries for Ubuntu, which don't require you (ideally) to compile them yourself. In this case you could simply clone this repository (or download the binary directly in the /bin/Ubuntu/) folder to your system and then start using it as usual:


## Build on a Linux machine

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

