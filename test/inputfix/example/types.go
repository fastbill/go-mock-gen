package example

import (
	"github.com/fastbill/go-httperrors/v2"
	"github.com/fastbill/go-mock-gen/test/inputnew/model"
)

// Exampler is an interface for the test.
type Exampler interface {
	FunctionZ(id int, user *model.StructA) (*model.StructB, *model.StructA, error)
	FunctionC(name, address string) *model.StructA
	FunctionD(user *model.StructA) *httperrors.HTTPError
}
