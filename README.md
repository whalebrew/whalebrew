# Whalebrew

Whalebrew creates aliases for Docker images so you can run them as if they were native commands. It's like Homebrew, but with Docker images.

Docker works well for packaging up development environments, but there are lots of tools that aren't tied to a particular project: `awscli` for managing your AWS account, `ffmpeg` for converting video, `wget` for downloading files, and so on. Whalebrew makes those things work with Docker, too.

    $ whalebrew install whalebrew/whalesay
    Unable to find image 'whalebrew/whalesay' locally
    Using default tag: latest
    latest: Pulling from whalebrew/whalesay
    c60055a51d74: Pull complete
    755da0cdb7d2: Pull complete
    969d017f67e6: Pull complete
    Digest: sha256:5f3a2782b400b2b23774709e0685d65b4493c6cbdb62fff6bbbd2a6bd393845b
    Status: Downloaded newer image for whalebrew/whalesay:latest
    🐳  Installed whalebrew/whalesay to /usr/local/bin/whalesay
    $ whalesay cool
     ______
    < cool >
     ------
       \
        \
         \
                       ##        .
                 ## ## ##       ==
              ## ## ## ##      ===
          /""""""""""""""""___/ ===
     ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
          \______ o          __/
           \    \        __/
             \____\______/


Whalebrew can run almost any CLI tool, but it isn't for everything (e.g. where commands must start instantly). It works particularly well for:

* **Complex dependencies.** For example, a Python app that requires C libraries, specific package versions, and other CLI tools that you don't want to clutter up your machine with.
* **Cross-platform portability.** Package managers tend to be very closely tied to the system they are running on. Whalebrew packages work on any modern version of macOS, Linux, and Windows (coming soon).

## Install

First, [install Docker](https://docs.docker.com/engine/installation/). The easiest way to do this on macOS is by installing [Docker for Mac](https://docs.docker.com/docker-for-mac/). You can install Docker for Mac via Homebrew:

    brew install --cask docker

Next, you can install whalebrew via Homebrew on macOS and Linux:

    brew install whalebrew

If you're not using Homebrew, you can download a binary and use that:

    curl -L "https://github.com/whalebrew/whalebrew/releases/download/0.4.1/whalebrew-$(uname -s)-$(uname -m)" -o /usr/local/bin/whalebrew; chmod +x /usr/local/bin/whalebrew

Windows support is theoretically possible, but not implemented yet.

## Usage

### Install packages

    $ whalebrew install whalebrew/wget

This will install the image `whalebrew/wget` as `/usr/local/bin/wget`.

The images in the `whalebrew` organization are a set of images that are known to work well with Whalebrew. You can also install any other images on Docker Hub too, but they may not work well:

    $ whalebrew install bfirsh/ffmpeg

### Find packages

    $ whalebrew search
    whalebrew/ack
    whalebrew/awscli
    whalebrew/docker-cloud
    whalebrew/ffmpeg
    whalebrew/gnupg
    ...

    $ whalebrew search wget
    whalebrew/wget

### List installed packages

    $ whalebrew list
    COMMAND     IMAGE
    ffmpeg      bfirsh/ffmpeg
    wget        whalebrew/wget
    whalebrew   whalebrew/whalebrew
    whalesay    whalebrew/whalesay

### Uninstall packages

    $ whalebrew uninstall wget

### Upgrade packages

To upgrade a single package, just pull its image:

    $ docker pull whalebrew/wget

## Configuration

Whalebrew reads configuration from either configuration files or environment variables.

The configuration file location can be specified using the `WHALEBREW_CONFIG_DIR` environment variable and defaults to `~/.whalebrew`.
The configuration file must be named `config.yaml`.

|Description|Default (if not specified anywhere)|Format in config files|Format in environment variables|
|-|-|-|-|
|The folder containing `config.yaml`|`~/.whalebrew`|N/A|`WHALEBREW_CONFIG_DIR=$HOME/my-config`|
|The directory to install packages in.|`/usr/local/bin`|`install_path: $HOME/.whalebrew/bin`|`WHALEBREW_INSTALL_PATH=$HOME/.whalebrew/bin`|

On a general basis, any configuration configured through environment variable will be prioritary compared to values from config files.

For example, if you have a whalebrew config of `install_path: $HOME/.whalebrew/bin` and an environment variable of `$HOME/.local/bin`, all whalebrew programs will be installed in `$HOME/.local/bin`.

### Configuration path lookup

Environment variables have precedence on any other value.
As soon as the `WHALEBREW_CONFIG_DIR` it defines the whalebrew installation directory.
When not set, whalebrew considers the first existing file between `~/.whalebrew/config.yaml`, `$XDG_CONFIG_HOME/whalebrew/config.yaml`, and for each `path` in `$XDG_DATA_DIRS`, whether `$path/whalebrew/config.yaml` exists

### Using custom registries

:warning: This feature is currently under development. Any feedback or comments are welcome!

Whalebrew now supports using several registries when searching for packages.

Each repository will be searched sequentially and matches on whalebrew packages will be shown, one per line.

To enable this feature, ensure you have a configuration file configured as defined [above](#configuration).

You can make one by running:

```
mkdir -p ${WHALEBREW_CONFIG_DIR:-~/.whalebrew}
cat > ${WHALEBREW_CONFIG_DIR:-~/.whalebrew}/config.yaml <<EOF
registries:
- dockerHub:
    owner: whalebrew
- dockerHub:
    owner: my-org
EOF
```

:warning: _Note_ that if you provide a single docker hub owner, only this owner will be searched for registries, replacing the default `whalebrew` docker hub organisation.

## How it works

Whalebrew is simple, and leans as much as possible on native Docker features:

* Packages are installed as files in `/usr/local/bin` (or a directory that you configure) with a [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) to make them executable. The content of the file is YAML that describes the options to pass to `docker run`, similar to a Compose service. For example:

        #!/usr/bin/env whalebrew
        image: whalebrew/whalesay

* When a package is executed, Whalebrew will run the specified image with Docker, mount the current working directory in `/workdir`, and pass through all of the arguments.

  To understand what it is doing, you can imagine it as a shell script that looks something like this:

      docker run -it -v "$(pwd)":/workdir -w /workdir $IMAGE "$@"

## Creating packages

Packages are Docker images published on Docker Hub. The requirements to make them work are:

* They must have the command to be run set as the entrypoint.
* They must only work with files in `/workdir`.

That's it. So long as your image is set up to work that way, it'll work with Whalebrew.

### Configuration

There are some labels you can use to configure how Whalebrew installs your image:

* `io.whalebrew.name`: The name to give the command. Defaults to the name of the image.
* `io.whalebrew.config.environment`: A list of environment variables to pass into the image from the current environment when the command is run. For example, putting this in your `Dockerfile` will pass through the values of `TERM` and `FOOBAR_NAME` in your shell when the command is run:

        LABEL io.whalebrew.config.environment '["TERM", "FOOBAR_NAME"]'

* `io.whalebrew.config.volumes`: A list of volumes to mount when the command is run. For example, putting this in your image's `Dockerfile` will mount `~/.docker` as `/root/.docker` in read-only mode:

        LABEL io.whalebrew.config.volumes '["~/.docker:/root/.docker:ro"]'

* `io.whalebrew.config.ports`: A list of host port to container port mappings to create when the command is run. For example, putting this in your image's `Dockerfile` will map container port 8100 to host port 8000:

        LABEL io.whalebrew.config.ports '["8100:8000"]'

* `io.whalebrew.config.networks`: A list of networks to connect the container to.

        LABEL io.whalebrew.config.networks '["host"]'

* `io.whalebrew.config.working_dir`: The path the working directory should be bound to in the container. For example putting this in your image's `Dockerfile` will ensure the working directory is available in /working_directory in the container

        LABEL io.whalebrew.config.working_dir '/working_directory'

* `io.whalebrew.config.keep_container_user`: Set this variable to true to keep the default container user. When set to true, whalebrew will not run the command as the current user using the docker `-u` flag

        LABEL io.whalebrew.config.keep_container_user 'true'

* `io.whalebrew.config.missing_volumes`: The behaviour to handle missing files or volumes into the container.

        LABEL io.whalebrew.config.missing_volumes 'skip'

  Possible values are
    * `error` to raise an error when trying to mount a non existing volume *this is the default behaviour*
    * `skip` to prevent binding the volume
    * `mount` to mount the volume anyway. This will result in docker [creating a host directory](https://docs.docker.com/engine/reference/commandline/run/#mount-volume--v---read-only)

* `io.whalebrew.required_version`: Specifies the minimum whalebrew version that is required to run the package. Examples: `<1.0.0`, `>0.1.0`, `>0.1.0 <1.0.0`

* `io.whalebrew.config.volumes_from_args`: A list of command line arguments of the program passed at runtime that must be considered and mounted as volumes:

        LABEL io.whalebrew.config.volumes_from_args '["-C", "--exec-path"]'

#### Using user environment variables

The labels `io.whalebrew.config.working_dir`, `io.whalebrew.config.volumes` and `io.whalebrew.config.environment` are expanded with user environment variables when the container is launched.

For example, if your image has this line in your `Dockerfile`:

        LABEL io.whalebrew.config.working_dir '$PWD'

At runtime, it will bind your working directory into the container at the same path and set it as the working directory.

#### Using hooks

In some cases, you might want to execute custom actions, like checking the integrity of the image or adding the whalebrew scripts to your whalebrew repository.
To do so, whalebrew will call git-like hooks when handling installation/uninstallation of a package.
Those hooks must be executable files located in `${WHALEBREW_CONFIG_DIR}/hooks`.

Whalebrew supports the following hooks:

|command & arguments|description|
|-|-|
|`pre-install ${DOCKER_IMAGE} ${EXECUTABLE_NAME}`|This hook is called before installing a package. If it fails, the whole installation process fails|
|`post-install ${EXECUTABLE_NAME}`|This hook is called after a package is installed. If it fails, the installation process fails, but the package is not uninstalled|
|`pre-uninstall ${EXECUTABLE_NAME}`|This hook is called before uninstalling a package. If it fails, the whole uninstallation process fails|
|`post-uninstall ${EXECUTABLE_NAME}`|This hook is called after a package is uninstalled. If it fails, the uninstallation process fails, but the package is not uninstalled|

### Whalebrew images

We maintain a set of packages which are known to follow these requirements under the `whalebrew` organization on [GitHub](https://github.com/whalebrew) and [Docker Hub](https://hub.docker.com/u/whalebrew/). If you want to add a package to this, open a pull request against [whalebrew-packages](https://github.com/whalebrew/whalebrew-packages).

## Thanks

* Justin Cormack for [the original idea](https://github.com/justincormack/dockercommand-cli) and generally just being very clever.
