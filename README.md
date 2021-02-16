opendex-launcher
================

[![Discord](https://img.shields.io/discord/628640072748761118.svg)](https://discord.gg/RnXFHpn)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

`opendex-launcher` is a lightweight binary launcher for [opendex-docker](https://github.com/opendexnetwork/opendex-docker), responsible for starting, stopping & updating containers, as well as orchestrating various flows, like creating an opendex environment. It is designed with stability in mind, not requiring frequent updates. It is embedded in [opendex-desktop](https://github.com/opendexnetwork/opendex-desktop) as well as [opendex-docker](https://github.com/opendexnetwork/opendex-docker). 
`opendex-launcher` also allows developers to run any branch of [opendex-docker](https://github.com/opendexnetwork/opendex-docker).

### Requirements

docker & docker-compose >= 18.09 (we recommend following the [official install instructions](https://docs.docker.com/get-docker/) to ensure compatibility)

### Build

On *nix platform
```sh
make
```

On Windows platform
```
mingw32-make
```

### Run

On *nix platform
```sh
export BRANCH=master
export NETWORK=mainnet
./opendex-launcher setup
```

On Windows platform (with CMD)
```
set BRANCH=master
set NETWORK=mainnet
./opendex-launcher setup
```

On Windows platform (with Powershell)
```
$Env:BRANCH = "master"
$Env:NETWORK = "mainnet"
./opendex-launcher setup
```
