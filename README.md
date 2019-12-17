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
go-mock-gen <filePath> <interfaceName> <mockStructName>
```
* Set `filePath` to the path to the file that contains the interface that should be mocked.
* Set `interfaceName` to the name of the interface that should be mocked.
* Set `mockStructName` to the name you want to give the mock struct that you will use in your tests.

#### Example
```
go-mock-gen ./pkg/repository/repository.go Persister Repository
```

### Updating an Existing Mock
```bash
go-mock-gen <filePath> <interfaceName> -f
```
* Set `filePath` to the path to the file that contains the interface that was mocked.
* Set `interfaceName` to the name of the interface for which the mock should be updated.
* ⚠️ The `-f` flag must be the last parameter.

#### Example
```
go-mock-gen ./pkg/repository/repository.go Persister -f
```

## Requirements to Use the Tool to Update Existings Mocks
Currently the following restrictions apply if you want to use this tool to update existing mocks.

* The mock is in a folder called `<packageName>mock` were package name is the name of the package of the interface.
* One of the comments contains the package name and interface name in the format `<packageName>.<interfaceName>`.
* The receiver variable is consistently called `m`.

## Example
### Creating a New Mock
Given the following interface
```go
type Exampler interface {
	FunctionA(user *model.StructA) (resultStr string, err error)
	FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error)
	FunctionC(name, address string, age int) *model.StructA
}
```
go-mock-gen will generate the following file
```go
package examplemock

import (
	"github.com/some-path/model"
	"github.com/stretchr/testify/mock"
)

// TestMock is a mock implementation of the example.Exampler interface.
type TestMock struct {
	mock.Mock
}

// FunctionA is a mock implementation of example.Exampler#FunctionA.
func (m *TestMock) FunctionA(user *model.StructA) (string, error) {
	args := m.Called(user)

	return args.String(0), args.Error(1)
}

// FunctionC is a mock implementation of example.Exampler#FunctionC.
func (m *TestMock) FunctionC(name string, address string, age int) *model.StructA {
	args := m.Called(name, address, age)

	if args.Get(0) != nil {
		return args.Get(0).(*model.StructA)
	}

	return nil
}

// FunctionZ is a mock implementation of example.Exampler#FunctionZ.
func (m *TestMock) FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error) {
	args := m.Called(id, user)

	if args.Get(0) != nil && args.Get(1) != nil {
		return args.Get(0).(*model.StructB), args.Get(1).(*model.StructA), args.Error(2)
	}

	return nil, nil, args.Error(2)
}

```

### Updating an Existing Mock
Given the interface above is changed to
```go
type Exampler interface {
	FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error)
	FunctionC(name, address string) *model.StructA
	FunctionD(user *model.StructA) error
}
```
and there is an existing mock with the following methods
```go
// FunctionA is a mock implementation of example.Exampler#FunctionA.
func (m *TestMock) FunctionA(user *model.StructA) (string, error) {
	args := m.Called(user)

	return args.String(0), args.Error(1)
}

// FunctionZ is a mock implementation of example.Exampler#FunctionZ.
func (m *TestMock) FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error) {
	args := m.Called(id, user)

	if args.Get(0) != nil {
		return args.Get(0).(*model.StructB), nil, args.Error(2)
	}

	if args.Get(1) != nil {
		return nil, args.Get(1).(*model.StructA), args.Error(2)
	}

	return nil, nil, args.Error(2)
}

// FunctionC is a mock implementation of example.Exampler#FunctionC.
func (m *TestMock) FunctionC(name string, address string, age int) *model.StructA {
	args := m.Called(name, address, age)

	if args.Get(0) != nil {
		return args.Get(0).(*model.StructA)
	}

	return nil
}
```
go-mock-gen will update these to
```go
// FunctionZ is a mock implementation of example.Exampler#FunctionZ.
func (m *TestMock) FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error) {
	args := m.Called(id, user)

	if args.Get(0) != nil {
		return args.Get(0).(*model.StructB), nil, args.Error(2)
	}

	if args.Get(1) != nil {
		return nil, args.Get(1).(*model.StructA), args.Error(2)
	}

	return nil, nil, args.Error(2)
}

// FunctionC is a mock implementation of example.Exampler#FunctionC.
func (m *TestMock) FunctionC(name string, address string) *model.StructA {
	args := m.Called(name, address)

	if args.Get(0) != nil {
		return args.Get(0).(*model.StructA)
	}

	return nil
}

// FunctionD is a mock implementation of example.Exampler#FunctionD.
func (m *TestMock) FunctionD(user *model.StructA) error {
	args := m.Called(user)

	return args.Error(0)
}
```
Notice that the customizations that were done in FunctionZ are kept in the update since the signiture of FunctionZ is still the same.

## Credit
Created with some code from [github.com/vektra/mockery](https://github.com/vektra/mockery).
