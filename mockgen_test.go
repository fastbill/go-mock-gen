package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindInterface(t *testing.T) {
	iface, err := findInterface("./test/inputnew/example/types.go", "Exampler")
	require.NoError(t, err)
	methods := iface.Methods()
	assert.Equal(t, 3, len(methods))
	assert.Equal(t, "FunctionA", methods[0].Name)
	assert.Equal(t, "FunctionC", methods[1].Name)
	assert.Equal(t, "FunctionZ", methods[2].Name)
}

func TestGenerateNewMock(t *testing.T) {
	cleanup(t)
	err := generateNewMock("./test/inputnew/example/types.go", "Exampler", "TestMock")
	require.NoError(t, err)
	expected, err := ioutil.ReadFile("./test/expectednew/result.go")
	require.NoError(t, err, "error in test setup")
	actual, err := ioutil.ReadFile("./test/inputnew/example/examplemock/testmock.go")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func TestUpdateMock(t *testing.T) {
	cleanup(t)
	err := copy.Copy("./test/inputfix/existing", "./test/inputfix/example/examplemock")
	require.NoError(t, err, "error in test setup")

	err = updateMock("./test/inputfix/example/types.go", "Exampler")
	require.NoError(t, err)

	expected, err := ioutil.ReadFile("./test/expectedfix/result.go")
	require.NoError(t, err, "error in test setup")
	actual, err := ioutil.ReadFile("./test/inputfix/example/examplemock/testmock.go")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func cleanup(t *testing.T) {
	err := os.RemoveAll("./test/inputnew/example/examplemock")
	assert.NoError(t, err, "error in test setup")

	err = os.RemoveAll("./test/inputfix/example/examplemock")
	assert.NoError(t, err, "error in test setup")
}
