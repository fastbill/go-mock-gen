package example

import "github.com/fastbill/go-mock-gen/testdata/inputnew/model"

// Exampler is an interface for the test.
type Exampler interface {
	FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error)
	FunctionC(name, address string) *model.StructA
	FunctionD(user *model.StructA) error
}
