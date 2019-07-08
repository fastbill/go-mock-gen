# Go-Mock-Gen [![Build Status](https://travis-ci.com/fastbill/go-mock-gen.svg?branch=master)](https://travis-ci.com/fastbill/go-mock-gen) [![Go Report Card](https://goreportcard.com/badge/github.com/fastbill/go-mock-gen)](https://goreportcard.com/report/github.com/fastbill/go-mock-gen)

## Description
This is a CLI tool to automatically create and update [testify mocks](https://github.com/stretchr/testify#mock-package).

In contrast to [github.com/vektra/mockery](https://github.com/vektra/mockery) it is not meant as a standard auto generation tool to run programatically. It is meant to produce readable code just as the developer would have written it. The tool just saves some time with the initial creation and the update of this code. Besides that the generated code should be treated just as hand-written code.

When updating existing mocks go-mock-gen only replaces the existing code in case the signature of the method has changed. Otherwise the mock method is not changed at all. This allows to modify the functions to adapt for special cases that might need to be addressed in the mock while still being able to effectifly use the tool in case methods have been added or signatures have been changed. The tool will also add and remove methods when they were added/removed from the interface.

⚠️ This tool should only be used together with a version control system like git and all changed the tool makes should be carefully reviewed before commiting them.

## Installation
```bash
go get github.com/fastbill/go-mock-gen
```

## Usage
### Creating a New Mock
```bash
go-mock-gen <folderPath> <interfaceName> <mockStructName>
```
* Set `folderPath` to the path to the folder where the interface can be found.
* Set `interfaceName` to the name of the interface that should be mocked.
* Set `mockStructName` to the name you want to give the mock struct that you will use in your tests.

#### Example
```
go-mock-gen ./pkg/repository Persister Repository
```

### Updating an Existing Mock
```bash
go-mock-gen <folderPath> <interfaceName> -f
```
* Set `folderPath` to the path to the folder where the interface can be found.
* Set `interfaceName` to the name of the interface that should be mocked.
* ⚠️ The `-f` flag must be the last parameter.

#### Example
```
go-mock-gen ./pkg/repository Persister -f
```

## Requirements to Use the Tool to Update Existings Mocks
Currently the following restrictions apply if you want to use this tool to update existing mocks.

* The mock is in a folder called `<packageName>mock` were package name is the name of the package of the interface.
* One of the comments contains the package name and interface name in the format `<packageName>.<interfaceName>`.
* The receiver variable is consistently called `m`.

Created with some code from [github.com/vektra/mockery](https://github.com/vektra/mockery).
