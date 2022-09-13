# Prometheus Exporter for IIO Sensors on the RFSoC.

This script is made to export IIO sensors values to Prometheus. Originally developed for CASPER's RFSoC. 

### Installation

All you need is to download the executable for your machine under GitHub releases. 
You can also compile it from source using the awesome cross-compilation capabilities of Go.

```shell
$ GOOS=linux GOARCH=arm64 go build
```

### Usage

Just run the executable inside the machine.