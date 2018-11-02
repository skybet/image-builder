# image-builder

Used to build multiple small docker images from a single git repository. The tool will clone the repo in memory and then calculate which images have change in the latest commit and then build those images before pushing them. The tool requires a certain directory structure to work e.g. 

```
./
hub.docker.com/
hub.docker.com/namespace/
my.private.repo.com/
my.private.repo.com/namespace1/
my.private.repo.com/namespace2/
```

It will create tags based on the commit hash of HEAD and the current branch name. For example if you had a Jenkins job which always built from the `release` branch you would end up with a tag called `release` which always has the latest code and also a tag like `abc0123` which refers to the current commit.

```
$ image-builder --help
Usage:
  image-builder [flags]

Flags:
      --config string        config file (default is $HOME/.image-builder.yaml)
  -d, --debug                Debug mode
  -u, --docker-host string   Docker host/socket (default "unix:///var/run/docker.sock")
  -a, --docker-auth string   Base64 encoded string (see below)
  -b, --git-branch string    Git branch to build (default "master")
  -g, --git-url string       Git repo to build
  -j, --json                 Log in json format
  -k, --key-path string      Path to private key
```

## Authentication

If your Docker repository requires authentication then you will need to pass in the docker-auth string.

To generate this, you'll need a username, a password and an email address:

`echo '{"username": "alice", "password":"MySecurePassword1!", "email":"alice@example.com"}' | base64`