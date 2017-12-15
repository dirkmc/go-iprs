go-iprs
===============================================

![](https://img.shields.io/badge/status-WIP-red.svg?style=flat-square)

> Go implementation of [IPRS spec](https://github.com/ipfs/specs/tree/master/iprs)

## Table of Contents

- [Status](#status)
- [Install](#install)
- [Usage](#usage)
- [License](#license)

## Status

Note: This module is a work in progress

During this process, you can check more about the state of this project on:

- [issues](https://github.com/dirkmc/go-iprs/issues)
- [libp2p specs](https://github.com/libp2p/specs)
- [IPRS spec](https://github.com/ipfs/specs/tree/master/iprs)


## Install

`go-iprs` is a standard Go module which can be installed with:

```sh
go get github.com/ipfs/go-iprs
```

Note that `go-iprs` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).

## Usage

IPRS Records are created with a `RecordValidity` and a `RecordSigner`. `RecordValidity` indicates under what conditions the record is considered valid, for example before a certain date (EOL) or between certain dates (TimeRange). `RecordSigner` adds verification data to a record, by signing it, eg with a private key, or with an x509 certificate. A factory is provided that helps construct these records from the `RecordValidity` and `RecordSigner`, with some methods for constructing common combinations (eg a record with an EOL that is signed with a private key)

### Examples

#### Creating an EOL record signed with a private key

```go
privateKey := GenerateAPrivateKey()
valueStore := CreateAValueStore()
ns := NewNameSystem(valueStore, 20)

// Create the record
f := NewRecordFactory(valueStore)
p := iprspath.IprsPath("/iprs/" + u.Hash(privateKey))
eol := time.Now().Add(time.Hour)
record = f.NewEolKeyRecord(path.Path("/ipfs/myIpfsHash"), privateKey, eol)

// Publish the record
record = f.NewEolKeyRecord(path.Path("/ipfs/myIpfsHash"), privateKey, eol)
err := ns.Publish(ctx, p, record)
if err != nil {
	fmt.Println(err)
}
```

#### Retrieving a record value

```go
iprsPath := GetIprsPath()
valueStore := CreateAValueStore()
ns := NewNameSystem(valueStore, 20)
val, err := ns.resolve(ctx, iprsPath)
if err == nil {
	fmt.Println(val)
}
```

### Using Gx and Gx-go

This module is packaged with [Gx](https://github.com/whyrusleeping/gx). In order to use it in your own project it is recommended that you:

```sh
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/ipfs/go-iprs
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.

## License

MIT
