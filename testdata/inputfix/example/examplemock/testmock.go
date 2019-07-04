package examplemock

import (
	"github.com/fastbill/go-mock-gen/testdata/inputnew/model"
	"github.com/stretchr/testify/mock"
)

// TestMock is a mock implementation of the example.Exampler interface.
type TestMock struct {
	mock.Mock
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

