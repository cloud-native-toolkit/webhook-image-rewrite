package util

import (
	"reflect"
)

func Any(collection interface{}, fn func (v interface{}) bool) bool {
	c := reflect.ValueOf(collection)

	for i := 0; i < c.Len(); i++ {
		if fn(c.Index(i).Interface()) {
			return true
		}
	}

	return false
}

func All(collection interface{}, fn func (v interface{}) bool) bool {
	c := reflect.ValueOf(collection)

	if c.Len() == 0 {
		return false
	}

	for i := 0; i < c.Len(); i++ {
		if !fn(c.Index(i).Interface()) {
			return false
		}
	}

	return true
}

func Map(collection interface{}, fn func(v interface{}) interface{}) interface{} {
	c := reflect.ValueOf(collection)

	if c.Len() == 0 {
		return make([]interface{}, 0)
	}

	tempResult := make([]interface{}, 0)

	for i := 0; i < c.Len(); i++ {
		val := fn(c.Index(i).Interface())

		tempResult = append(tempResult, val)
	}

	tmpVal := tempResult[0]
	typ := reflect.TypeOf(tmpVal)

	result := reflect.MakeSlice(reflect.SliceOf(typ), 0, (c.Cap() + 1) * 2)

	for i := 0; i < len(tempResult); i++ {
		result = reflect.Append(result, reflect.ValueOf(tempResult[i]))
	}

	return result.Interface()
}

func Filter(collection interface{}, fn func(v interface{}) bool) interface{} {
	c := reflect.ValueOf(collection)
	typ := reflect.TypeOf(collection).Elem()

	result := reflect.MakeSlice(reflect.SliceOf(typ), 0, (c.Cap() + 1) * 2)

	for i := 0; i < c.Len(); i++ {
		if fn(c.Index(i).Interface()) {
			result = reflect.Append(result, c.Index(i))
		}
	}

	return result.Interface()
}

func Index(collection interface{}, test interface{}) int {
	c := reflect.ValueOf(collection)

	for i := 0; i < c.Len(); i++ {
		if c.Index(i).Interface() == test {
			return i
		}
	}

	return -1
}

func Includes(collection interface{}, test interface{}) bool {
	return Index(collection, test) >= 0
}
