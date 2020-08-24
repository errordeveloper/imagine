# `imagine` - a high-level container image build tool

It is a (slightly) opinionated tool for bulding images with `docker buildx bake`,
it currently implements a few basic commands.

Things `imagine` has opinions about:

- image tagging (based on git)
- image testing as a separate target
- registry as a separate notion to image name and tag with multi-registry support
- by default it will not overwrite existing tags, neither it will rebuild

## How it works?

### Main commands

- `imagine build` – will build the image and (optionally) push it all specified registries
- `imagine image` – only writes image tags to stdout
  - it supports a relevant subset of `imagine build` flags
- `imagine generate` – will writes buildx manifest to stdout 
  - it supports a relevant subset of `imagine build` flags

### Tagging and Rebuilding

`imagine` has two tagging modes:

   - git revision or semver tag - used when imege build is defined by the entier repository
     - this mode is enabled with `--root`
     - only semver git tags are recognised, any non-semver tags are ignored
     - when multiple git tag point to the same commit, the highest version is picked
   - git tree hash - used when image build is defined by a subdirecotry

At present git tree hash format is a full-lenght SHA1, while git revision is a short SHA1
(`git rev-parse --short`). This may change in the future, it may also be possible to pass
tag prefix or a custom image tag.

When there changes to any of the checked-in files, `-wip` suffix is appended.  When the build
is not a base branch (can be set with `--base-brach` and defaults to `master`), a `-dev` suffix
is appended. This behaviour can be controlled with `--without-tag-suffix`.

Images are rebuilt only when there is no remote image in at least one of the given registries.
With git revsion tagging mode this means only new revisions are re-built, and with git tree
hash mode it means that new images are built only whenever there are changes to the given
subdirectory that defines the image.

A rebuild can be force with `--force`, or when either of the suffices (`-dev` and/or `-wip`)
had been appended to the image.

### Testing

If you have tests defined in `FROM ... as test` section of your `Dockerfile`, you can use
`--test` flag to run those tests.

### Examples

First, you need to make sure to setup a BuildKit instance:
```
builder="$(docker buildx create)"
```

You can use any pre-existing BuildKit instance (check `docker buildx ls`), but you cannot use 
the default `docker` driver, as it only suppors a limited set of `buildx` features.

And, pick your username, e.g.:
```
username=errordeveloper
```


To build an image that takes `examples/alpine` subdircorey as input, run:

```
imagine build \
  --builder "${builder}" \
  --registry "docker.io/${username}" \
  --registry "quay.io/${username}" \
  --name imagine-alpine-example \
  --base ./examples/alpine \
  --cleanup
```

To build an image that is defined by entier repository, run:
```
imagine build \
  --builder "${builder}" \
  --registry "docker.io/${username}" \
  --registry "quay.io/${username}" \
  --name imagine-imagine-example \
  --root \
  --base ./ \
  --dockerfile ./examples/imagine/Dockerfile
  --cleanup
```
