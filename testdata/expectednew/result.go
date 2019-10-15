package examplemock

import (
	"github.com/fastbill/go-mock-gen/v2/testdata/inputnew/model"
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
