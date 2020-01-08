# Azure Storage Blob SDK for Go
[![GoDoc Widget]][GoDoc] [![Build Status][Travis Widget]][Travis]

The Microsoft Azure Storage SDK for Go allows you to build applications that takes advantage of Azure's scalable cloud storage. 

This repository contains the open source Blob SDK for Go. The [File SDK][File SDK] and [Queue SDK][Queue SDK] are also available.

## Features
* Blob Storage
	* Create/List/Delete Containers
	* Create/Read/List/Update/Delete Block Blobs
	* Create/Read/List/Update/Delete Page Blobs
	* Create/Read/List/Update/Delete Append Blobs

## Getting Started
* If you don't already have it, install [the Go distribution](https://golang.org/dl/)
* Get the SDK, with any method you prefer:
    * Go Get: ```go get github.com/Azure/azure-storage-blob-go/azblob```
    * Dep: add ```github.com/Azure/azure-storage-blob-go``` to Gopkg.toml:
        ```
        [[constraint]]
          version = "0.3.0"
          name = "github.com/Azure/azure-storage-blob-go"
        ```
    * Module: simply import the SDK and Go will download it for you
* Use the SDK:
```import "github.com/Azure/azure-storage-blob-go/azblob"```

## Version Table
* If you are looking to use a specific version of the Storage Service, please refer to the following table: 

| Service Version | Corresponding SDK Version | Import Path                                              |
|-----------------|---------------------------|----------------------------------------------------------|
| 2016-05-31      | 0.2.0                     | github.com/Azure/azure-storage-blob-go/2016-05-31/azblob |
| 2017-07-29      | 0.2.0                     | github.com/Azure/azure-storage-blob-go/2017-07-29/azblob |
| 2018-03-28      | 0.3.0 - 0.5.0             | github.com/Azure/azure-storage-blob-go/azblob            |
| 2018-11-09      | 0.6.0 - 0.7.0             | github.com/Azure/azure-storage-blob-go/azblob            |

Note: the directory structure of the SDK has changed dramatically since 0.3.0. The different Service Versions are no longer sub-directories;
the latest `azblob` is directly under the root directory. In the future, each new Service Version will be introduced with a new major semantic version.
		
## SDK Architecture

* The Azure Storage SDK for Go provides low-level and high-level APIs.
	* ServiceURL, ContainerURL and BlobURL objects provide the low-level API functionality and map one-to-one to the [Azure Storage Blob REST APIs](https://docs.microsoft.com/en-us/rest/api/storageservices/blob-service-rest-api)
	* The high-level APIs provide convenience abstractions such as uploading a large stream to a block blob (using multiple PutBlock requests).

## Code Samples
* [Blob Storage Examples](https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#pkg-examples)

## License
This project is licensed under MIT.

## Contributing
This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

[GoDoc]: https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob
[GoDoc Widget]: https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob?status.svg
[Travis Widget]: https://travis-ci.org/Azure/azure-storage-blob-go.svg?branch=master
[Travis]: https://travis-ci.org/Azure/azure-storage-blob-go
[File SDK]: https://github.com/Azure/azure-storage-file-go
[Queue SDK]: https://github.com/Azure/azure-storage-queue-go
