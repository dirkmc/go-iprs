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

IPRS Records are created with a [RecordValidity](https://github.com/dirkmc/go-iprs/blob/master/record/record.go#L20) and a [RecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/record.go#L33). `RecordValidity` indicates under what conditions the record is considered valid, for example before a certain date ([EOL](https://github.com/dirkmc/go-iprs/blob/master/record/eol.go)) or between certain dates ([TimeRange](https://github.com/dirkmc/go-iprs/blob/master/record/range.go)). `RecordSigner` adds verification data to a record, by signing it, eg with a [private key](https://github.com/dirkmc/go-iprs/blob/master/record/key.go), or with an [x509 certificate](https://github.com/dirkmc/go-iprs/blob/master/record/cert.go). A [factory](https://github.com/dirkmc/go-iprs/blob/master/record/factory.go) is provided that helps construct these records from the `RecordValidity` and `RecordSigner`, with some methods for constructing common combinations (eg a record with an EOL that is signed with a private key)

### Examples

Records created with the [KeyRecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/key.go) have a `BasePath()` at `/iprs/<key hash>`

#### Creating an EOL record signed with a private key

```go
privateKey := GenerateAPrivateKey()
valueStore := CreateAValueStore()
rs := NewRecordSystem(valueStore, 20)

// Create the record
f := NewRecordFactory(valueStore)
eol := time.Now().Add(time.Hour)
record = f.NewEolKeyRecord(path.Path("/ipfs/myIpfsHash"), privateKey, eol)

// Publish the record
iprsKey, err := record.BasePath() // /iprs/<key hash>
if err != nil {
	fmt.Println(err)
}
err = rs.Publish(ctx, iprsKey, record)
if err != nil {
	fmt.Println(err)
}
```

#### Creating a TimeRange record signed with a private key

```go
privateKey := GenerateAPrivateKey()
valueStore := CreateAValueStore()
rs := NewRecordSystem(valueStore, 20)

// Create the record
f := NewRecordFactory(valueStore)
var BeginningOfTime *time.Time // nil indicates the beginning of time (ie, no start date)
var EndOfTime *time.Time       // nil indicates the end of time (ie, no expiration)
start := BeginningOfTime
end := time.Now().Add(time.Hour)
record = f.NewRangeKeyRecord(path.Path("/ipfs/myIpfsHash"), privateKey, start, &end)

// Publish the record
iprsKey, err := record.BasePath() // /iprs/<key hash>
if err != nil {
	fmt.Println(err)
}
err = rs.Publish(ctx, iprsKey, record)
if err != nil {
	fmt.Println(err)
}
```

Records created with the [CertRecordSigner](https://github.com/dirkmc/go-iprs/blob/master/record/cert.go) have a `BasePath()` at `/iprs/<ca cert key hash>` and can append an arbitrary sub path onto the end of it, eg `/iprs/<ca cert key hash>/mypath/mystuff`. The CA Certificate can then issue a child certificate that can be used to create a record under the CA Certificate's path. This provides a way to share IPRS path ownership between different users. For example Alice creates a CA Certificate and publishes a record at `/iprs/<alice ca cert hash>/alice/repos/cool/project`. She then issues a child certificate to Bob. Bob can now publish a new record to the same IPRS key.

#### Creating an EOL record signed with a CA certificate key

```go
caCert, caPk := GenerateCACertificate()
childCert, childPk := GenerateChildCertificate(caCert, caPk)
valueStore := CreateAValueStore()
rs := NewRecordSystem(valueStore, 20)

// Create the record with the CA certificate
f := NewRecordFactory(valueStore)
eol := time.Now().Add(time.Hour)
// Value is IPFS path of Alice's commit
record = f.NewEolCertRecord(path.Path("/ipfs/ipfsHashOfAlicesCommit"), caCert, caPk, eol)

// Publish the record
iprsKey, err := record.BasePath() + "/alice/repos/cool/project" // /iprs/<key hash>/alice/repos/cool/project
if err != nil {
	fmt.Println(err)
}
err = rs.Publish(ctx, iprsKey, record)
if err != nil {
	fmt.Println(err)
}

// Create a record with the child certificate
eol := time.Now().Add(time.Hour)
// Value is IPFS path of Bob's commit
record2 = f.NewEolCertRecord(path.Path("/ipfs/ipfsHashOfBobsCommit"), childCert, childPk, eol)

// Publish the record to the same IPRS path
// /iprs/<key hash>/alice/repos/cool/project
err = rs.Publish(ctx, iprsKey, record2)
if err != nil {
	fmt.Println(err)
}
```

#### Retrieving a record value

```go
iprsPath := GetIprsPath()
valueStore := CreateAValueStore()
rs := NewRecordSystem(valueStore, 20)
val, err := rs.resolve(ctx, iprsPath)
if err == nil {
	fmt.Println(val)
}
```

### Validators

IPRS provides a validator and selector for the `/iprs/` path at [validation.RecordChecker](https://github.com/dirkmc/go-iprs/blob/master/validation/validation.go). There is also a validator and selector for the `/cert/` path (for x509 certificates) at [certificate.ValidateCertificateRecord](https://github.com/dirkmc/go-iprs/blob/master/certificate/validator.go) and [certificate.CertificateSelector](https://github.com/dirkmc/go-iprs/blob/master/certificate/validator.go)

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
