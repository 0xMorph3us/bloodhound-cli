# BloodHound CLI

![Go](https://img.shields.io/github/go-mod/go-version/SpecterOps/bloodhound-cli?color=50B071)

![GitHub Release (Latest by Date)](https://img.shields.io/github/v/release/SpecterOps/bloodhound-cli?label=Latest%20Release&color=E61616
)
![GitHub Release Date](https://img.shields.io/github/release-date/SpecterOps/bloodhound-cli?label=Release%20Date&color=E1E2EF)

![BHCLI.png](BHCLI.png)

Golang code for the `bloodhound-cli` binary in [BloodHound](https://github.com/SpecterOps/BloodHound). This binary provides control for various aspects of BloodHound's configuration.

BloodHound CLI is compatible with Docker Compose v2 and Podman. If using Podman, configure [Docker compatibility mode](https://podman-desktop.io/docs/migrating-from-docker/managing-docker-compatibility).

## Usage

Execute `./bloodhound-cli help` for usage information (see below). 

More information about BloodHound and how to manage it with `bloodhound-cli` can be found on the [BloodHound Community Edition Quickstart Guide](https://bloodhound.specterops.io/get-started/quickstart/community-edition-quickstart), which is part of the [BloodHound documentation](https://bloodhound.specterops.io/home).

## Compilation

Releases are compiled with the following command to set version and build date information:

```bash
make
```

The version for rolling releases is set to `rolling`.

You can also use the Makefile:

```bash
make install   # download Go dependencies
make           # build bloodhound-cli
```
