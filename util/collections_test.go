package util

import (
	"strings"
	"testing"
)

func Test_Any_onematch_true(t *testing.T) {
	collection := []string{"one", "two", "three"}
	matchValue := "one"

	fn := func (val interface{}) bool {
		return val.(string) == matchValue
	}

	result := Any(collection, fn)

	if !result {
		t.Error("A value in the collection should match '" + matchValue + "'")
	}
}

func Test_Any_onematch_false(t *testing.T) {
	collection := []string{"one", "two", "three"}
	matchValue := "four"

	fn := func (val interface{}) bool {
		return val == matchValue
	}

	result := Any(collection, fn)

	if result {
		t.Error("A value in the collection should not match '" + matchValue + "'")
	}
}

func Test_All_true(t *testing.T) {
	collection := []string{"one", "one", "one"}
	matchValue := "one"

	fn := func (val interface{}) bool {
		return val == matchValue
	}

	result := All(collection, fn)

	if !result {
		t.Error("All values in the collection should match '" + matchValue + "'")
	}
}

func Test_All_false(t *testing.T) {
	collection := []string{"one", "one", "three"}
	matchValue := "one"

	fn := func (val interface{}) bool {
		return val == matchValue
	}

	result := All(collection, fn)

	if result {
		t.Error("One value in the collection should not match '" + matchValue + "'")
	}
}

type TestOwner struct {
	Val TestVal
}

type TestVal struct {
	Name string
	Value string
}

func Test_Map_string(t *testing.T) {
	collection := []TestVal{
		{
			Name: "test1",
			Value: "value1",
		},
		{
			Name: "test2",
			Value: "value2",
		},
	}

	mapper := func (val interface{}) interface{} {
		v := val.(TestVal)

		result := v.Value

		return result
	}

	var result []string
	result = Map(collection, mapper).([]string)

	if result[0] != "value1" {
		t.Error("The first value should be value1")
	}
}

func Test_Map_TestVal(t *testing.T) {
	collection := []TestOwner{
		TestOwner{Val: TestVal{
			Name: "test1",
			Value: "value1",
		}},
		TestOwner{Val: TestVal{
			Name: "test2",
			Value: "value2",
		}},
	}

	mapper := func (val interface{}) interface{} {
		v := val.(TestOwner)

		var result TestVal
		result = v.Val

		return result
	}

	var result []TestVal
	result = Map(collection, mapper).([]TestVal)

	if result[0] != collection[0].Val {
		t.Error("The first value should be value1")
	}
}

func Test_Filter_strings(t *testing.T) {
	collection := []string{"val1", "val2", "val3", "test"}

	filterVal := func (val interface{}) bool {
		s := val.(string)

		return strings.HasPrefix(s, "val")
	}

	var result []string
	result = Filter(collection, filterVal).([]string)

	if len(result) != 3 {
		t.Error("Should have 3 filtered values")
	}

	if result[0] != "val1" {
		t.Error("Should have val1 as first value")
	}

	if result[1] != "val2" {
		t.Error("Should have val2 as second value")
	}

	if result[2] != "val3" {
		t.Error("Should have val3 as third value")
	}
}
