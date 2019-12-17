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
	iface, err := findInterface("./testdata/inputnew/example/types.go", "Exampler")
	require.NoError(t, err)
	assert.Equal(t, 3, iface.Type.NumMethods())
	assert.Equal(t, "FunctionA", iface.Type.Method(0).Name())
	assert.Equal(t, "FunctionC", iface.Type.Method(1).Name())
	assert.Equal(t, "FunctionZ", iface.Type.Method(2).Name())
}

func TestGenerateNewMock(t *testing.T) {
	cleanup(t)
	err := generateNewMock("./testdata/inputnew/example/types.go", "Exampler", "TestMock")
	require.NoError(t, err)
	expected, err := ioutil.ReadFile("./testdata/expectednew/result.go")
	require.NoError(t, err, "error in test setup")
	actual, err := ioutil.ReadFile("./testdata/inputnew/example/examplemock/testmock.go")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func TestUpdateMock(t *testing.T) {
	cleanup(t)
	err := copy.Copy("./testdata/inputfix/existing", "./testdata/inputfix/example/examplemock")
	require.NoError(t, err, "error in test setup")

	err = updateMock("./testdata/inputfix/example/types.go", "Exampler")
	require.NoError(t, err)

	expected, err := ioutil.ReadFile("./testdata/expectedfix/result.go")
	require.NoError(t, err, "error in test setup")
	actual, err := ioutil.ReadFile("./testdata/inputfix/example/examplemock/testmock.go")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func cleanup(t *testing.T) {
	err := os.RemoveAll("./testdata/inputnew/example/examplemock")
	assert.NoError(t, err, "error in test setup")

	err = os.RemoveAll("./testdata/inputfix/example/examplemock")
	assert.NoError(t, err, "error in test setup")
}
