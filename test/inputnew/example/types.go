package example

import "github.com/fastbill/go-mock-gen/test/inputnew/model"

// Exampler is an interface for the test.
type Exampler interface {
	FunctionA(user *model.StructA) (resultStr string, err error)
	FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error)
	FunctionC(name, address string, age int) *model.StructA
}
