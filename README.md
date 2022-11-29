# docker-registry-cleaner
a Go application to remove images from a remote Docker Registry

 [![Lint Status](https://github.com/prune998/docker-registry-cleaner/workflows/Golangci-lint/badge.svg)](https://github.com/prune998/docker-registry-cleaner/actions)
 [![Lint Status](https://github.com/prune998/docker-registry-cleaner/workflows/GoBuild/badge.svg)](https://github.com/prune998/docker-registry-cleaner/actions)

## Usage

```bash
./docker-registry-cleaner  -project=kube00-XXX -registryURL="https://us.gcr.io" -username='_token' -password=$(gcloud auth print-access-token) -logLevel=info -filter="^kube00-XXX/us.gcr.io/imagename/imagename2:1234"
```

- registry URL: the URL to reach the registry. Do not add the `/v2` endpoint
  - Google: `https://us.gcr.io` for the US registry. Can also be `https://gcr.io`
- project: the name of the project (keep empty is no project is used)
  - Google: find your project with `gcloud projects list`
- username: your Registry username.
  - Google: `_token`
- password: well, your password
  - Google: get your token password with `gcloud auth print-access-token`
- logLevel: one of debug, info, warning, error. When using `debug` you will also get the HTTP calls to the Registry.
- filter: a RegExp to filter which images to list/delete. Leave empty to list/delete all images.
- delete: when set, will delete the manifest (the image) from the registry.
- check: when set, search for the image in the repository and exit with 0 (found) or 1 (not found)
- deleteUntagged: delete images from a repository if they have no tags. Only work when the filter contains `.*` as tag version

### Use Case

1. Delete a tagged image: use a filter like `project/image:v1.2.3`
1. Delete all images that have a tag: use a filter like `project/image:.*`
1. Delete all images that have a tag AND the others without tags: use a filter like `project/image:.*` AND `-deleteUntagged` option

### Manual test
  As a reminder, you can also do all this by hand using `curl`. If you're on Google Container Registry, you can do :

  ```bash
  curl -ks -u "oauth2accesstoken:$(gcloud auth print-access-token)" https://us.gcr.io/v2/project/project-name/tags/list | jq '.'
  ```

## Build

You need Go v1.19 or newer to build. 

```bash
go build 
```

## status

### 20221129-1

Removed vendors, updated to run with Go 1.19

### 20200116-1
It seems the SHA256 part of the manifest is needed, at least for Google Container registry...No error is returned but it does nothing if sha256 is not there...
Also, when deletting tags, we delete the image. This will silentely fail if there are other tags that are not deleted. I introduced a new option `deleteUntagged` so it will also remove images without tags when deleting `.*`.

### 20180608-2

Turn out the digest is like `sha256:b618c166f0b066dd9bba7...` while you only need the hash to delete the image (`b618c166f0b066dd9bba7...`)
After code update, I have a 202 from the registry... but the image is in fact never deleted (at least on Google GCR)
  
### 20180608-1

As of now this tool is working to list images from a Google registry.
It does not work to remove images. The error is :

```bash
Delete https://us.gcr.io/v2/kube00-XXXX/imagename/imagename2/manifests/sha256:5104db36afd2ea4a3977174e8ee1ce0fcec5678401a50d1a1cbcf240f2fd7da2:
http: non-successful response
status=404
body={
    "errors":[
        {"code":"NAME_UNKNOWN",
         "message":"Failed to compute blob liveness for manifest: 'sha256:5104db36afd2ea4a3977174e8ee1ce0fcec5678401a50d1a1cbcf240f2fd7da2'"
         }]
    }
```
