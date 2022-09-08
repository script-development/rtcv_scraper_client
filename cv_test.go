package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func mustEq(a string, b string) {
	if a != b {
		panic(fmt.Sprintf("left != right : %s != %s", a, b))
	}
}

func TestStrippedCVWithOriginal(t *testing.T) {
	testInput := []byte(`{"referenceNumber":"a","other":true,"personalDetails":{"zip":"1234"}}`)
	testOutput := StrippedCVWithOriginal{}
	err := json.Unmarshal(testInput, &testOutput)
	checkErr(err)
	mustEq("a", testOutput.ReferenceNumber)
	mustEq("1234", testOutput.PersonalDetails.Zip)

	formattedOutput, err := json.Marshal(testOutput)
	checkErr(err)
	mustEq(string(testInput), string(formattedOutput))

	testInput = []byte(strings.Join([]string{`[`,
		`{"referenceNumber":"a","other1":true,"personalDetails":{"zip":"1234"}},`,
		`{"referenceNumber":"b","personalDetails":{"zip":"4321","other2":true}}`,
		`]`}, ""))
	testOutputs := []StrippedCVWithOriginal{}
	err = json.Unmarshal(testInput, &testOutputs)
	checkErr(err)
	mustEq("a", testOutputs[0].ReferenceNumber)
	mustEq("1234", testOutputs[0].PersonalDetails.Zip)
	mustEq("b", testOutputs[1].ReferenceNumber)
	mustEq("4321", testOutputs[1].PersonalDetails.Zip)

	formattedOutput, err = json.Marshal(testOutputs)
	checkErr(err)
	mustEq(string(testInput), string(formattedOutput))
}
