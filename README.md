go-iprs
===============================================

![](https://img.shields.io/badge/status-WIP-red.svg?style=flat-square)

> Go implementation of [IPRS spec](https://github.com/ipfs/specs/tree/master/iprs). Note: This module is a work in progress

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [License](#license)

## Install

`go-iprs` is a standard Go module which can be installed with:

```sh
go get github.com/ipfs/go-iprs
```

Note that `go-iprs` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).

## Usage

IPRS Records are published to a path that consists of a validation CID (eg CID of a public key) and an ID, eg `/iprs/<cid>/photos`

The record value is the path to an IPLD node in the block store. It can be
- an IPFS path eg `/ipfs/<B58hash>/some/path`
- the raw bytes of a CID pointing to an IPLD node
- a CID in string format with an optional path, eg `<cid>/some/path`
- an IPNS path eg `/ipns/<B58hash>/some/path` or `/ipns/ipfs.io/some/path`
- an IPRS path eg `/iprs/<cid>/photos/3/size` or `/iprs/ipfs.io/some/path`

Records are created with a [RecordValidation](https://github.com/dirkmc/go-iprs/blob/master/record/record.go#L17) and a [RecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/record.go#L32). `RecordValidation` indicates under what conditions the record is considered valid, for example before a certain date ([EOL](https://github.com/dirkmc/go-iprs/blob/master/record/eol.go)) or between certain dates ([TimeRange](https://github.com/dirkmc/go-iprs/blob/master/record/range.go)). `RecordSigner` adds verification data to a record, by signing it, eg with a [private key](https://github.com/dirkmc/go-iprs/blob/master/record/key.go), or with an [x509 certificate](https://github.com/dirkmc/go-iprs/blob/master/record/cert.go).

### Examples

Records created with the [KeyRecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/key.go) have a `BasePath()` at `/iprs/<key hash>`

#### Creating an EOL record signed with a private key

```go
pk := GenerateAPrivateKey()
vstore := CreateAValueStore()
dag := CreateADAGStore()
rs := NewRecordSystem(vstore, dag, rsv.NoCacheOpts)

// Get the CID of the target Node
p1, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
if err != nil {
	return err
}

// Create the record
eol := time.Now().Add(time.Hour)
validation := rec.NewEolRecordValidation(eol)
signer := rec.NewKeyRecordSigner(pk)
record, err := rec.NewRecord(validation, signer, p1.Bytes())
if err != nil {
	return err
}
iprsKey, err := signer.BasePath("myrec")
if err != nil {
	return err
}

// Publish the record
err = rs.Publish(ctx, iprsKey, record)
```

#### Creating a TimeRange record signed with a private key

```go
// Create the record
var BeginningOfTime *time.Time // nil indicates the beginning of time (ie, no start date)
var EndOfTime *time.Time       // nil indicates the end of time (ie, no expiration)
start := BeginningOfTime
end := time.Now().Add(time.Hour)
validation := rec.NewRangeRecordValidation(start, end)
signer := rec.NewKeyRecordSigner(pk)
record, err := rec.NewRecord(validation, signer, p1.Bytes())
if err != nil {
	return err
}
iprsKey, err := signer.BasePath("myrec")
if err != nil {
	return err
}

// Publish the record
err = rs.Publish(ctx, iprsKey, record)
```

Records created with the [CertRecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/cert.go) have a `BasePath()` at `/iprs/<ca cert key hash>`. The CA Certificate can issue a child certificate that can be used to create a record under the CA Certificate's path. This provides a way to share IPRS path ownership between different users. For example Alice creates a CA Certificate and publishes a record at `/iprs/<alice ca cert hash>/myrepo`. She then issues a child certificate to Bob. Bob can now publish a new record to the same IPRS key.

#### Creating an EOL record signed with a CA certificate key

```go
caCert, caPk := GenerateCACertificate()
childCert, childPk := GenerateChildCertificate(caCert, caPk)

// Value is CID of Alice's commit
p1, err := cid.Parse("/ipfs/ipfsHashOfAlicesCommit")

// Create the record with the CA certificate
validation := rec.NewEolRecordValidation(eol)
signer := rec.NewCertRecordSigner(caCert, caPk)
record, err := rec.NewRecord(validation, signer, p1.Bytes())
if err != nil {
	return err
}

// Publish the record
iprsKey, err := signer.BasePath("myrepo") // /iprs/<cert hash>/myrepo
if err != nil {
	return err
}
err = rs.Publish(ctx, iprsKey, record)
if err != nil {
	return err
}

// ...

// Create a record with the child certificate
// Value is CID of Bob's commit
p2, err := cid.Parse("/ipfs/ipfsHashOfBobsCommit")

validation := rec.NewEolRecordValidation(eol)
signer := rec.NewCertRecordSigner(childCert, childPk)
record2, err := rec.NewRecord(validation, signer, p2.Bytes())
if err != nil {
	return err
}

// Publish the record to the same IPRS path
// /iprs/<cert hash>/myrepo
err = rs.Publish(ctx, iprsKey, record2)
```

#### Resolving an IPRS path to its target Node

```go
iprsPath := GetIprsPath()
nodeLink, path, err := rs.Resolve(ctx, iprsPath)
fmt.Printf("Link with CID %s and path %s", nodeLink.Cid, path)
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
