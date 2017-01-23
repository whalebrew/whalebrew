# Whalebrew

Whalebrew creates aliases for Docker images so you can run them as if they were native commands. It's like Homebrew, but with Docker images.

    $ whalebrew install whalebrew/whalesay
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

## Install

On macOS and Linux:

    curl -L "https://github.com/whalebrew/whalebrew/releases/download/0.0.1/whalebrew" -o /usr/local/bin/whalebrew; chmod +x /usr/local/bin/whalebrew

Windows support is theoretically possible, but not implemented yet.

## Usage

### Install packages

    $ whalebrew install whalebrew/wget

This will install the image `whalebrew/wget` as `/usr/local/bin/wget`.

The images in the `whalebrew` organization are a set of images that are known to work well with Whalebrew. You can also install any other images on Docker Hub too, but they may not work well:

    $ whalebrew install bfirsh/ffmpeg

### Find packages

    $ whalebrew search wget
    whalebrew/wget

### List installed packages

    $ whalebrew list
    COMMAND     IMAGE
    ffmpeg      bfirsh/ffmpeg
    wget        whalebrew/wget
    whalebrew   whalebrew/whalebrew
    whalesay    whalebrew/whalesay

### Uninstall packages

    $ whalebrew uninstall wget

### Upgrade packages

Upgrade all packages:

    $ whalebrew upgrade

To upgrade a single package, just pull its image:

    $ docker pull whalebrew/wget

### Upgrade Whalebrew

    $ whalebrew upgrade whalebrew

(Did I mention Whalebrew is a Whalebrew package?)

## Configuration

Whalebrew is configured with environment variables, which you can either provide at runtime or put in your `~/.bashrc` file (or whatever shell you use).

 - `WHALEBREW_INSTALL_PATH`: The directory to install packages in. (default: `/usr/bin/local`)

## How it works

Whalebrew is very simple, and leans as much as possible on native Docker features:

* Packages are installed as files in `/usr/local/bin` (or a directory that you configure) with a [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) to make them executable. The content of the file is YAML that describes the options to pass to `docker run`, similar to a Compose service. For example:

        #!/usr/bin/env whalebrew run
        image: whalebrew/whalesay

* When a package is executed, Whalebrew will run the specified image with Docker, mount the current working directory in `/workdir`, and pass through all of the arguments.

  To understand what it is doing, you can imagine it as a shell script that looks something like this:

      docker run -it -v "$(pwd)":/workdir -w /workdir $IMAGE "$@"

## Creating packages

Packages are Docker images published on Docker Hub. The requirements to make them work are:

* They must have the command to be run set as the entrypoint.
* They must only work with files in `/workdir`.

That's it. So long as your image is set up to work that way, it'll work with Whalebrew.

We maintain a set of packages which are known to follow these requirements under the `whalebrew` organization on [GitHub](https://github.com/whalebrew) and Docker Hub. If you want to add a package to this, open an issue.

## Thanks

* Justin Cormack for [the original idea](https://github.com/justincormack/dockercommand-cli) and generally just being very clever.
