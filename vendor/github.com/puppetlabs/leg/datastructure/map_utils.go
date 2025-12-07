package datastructure

import (
	"reflect"
)

func mapGetInto(m Map, key, into interface{}) bool {
	value, found := m.Get(key)

	if found {
		target := reflect.ValueOf(into).Elem()
		target.Set(coalesceInvalidToZeroValueOf(reflect.ValueOf(value), target.Type()))
	}

	return found
}

func mapKeys(m Map) []interface{} {
	keys := make([]interface{}, m.Size())

	i := 0
	m.ForEach(func(key, value interface{}) error {
		keys[i] = key
		i++

		return nil
	})

	return keys
}

func mapKeysInto(m Map, into interface{}) {
	p := reflect.ValueOf(into).Elem()
	pt := p.Type().Elem()

	slice := p

	m.ForEach(func(key, value interface{}) error {
		v := coalesceInvalidToZeroValueOf(reflect.ValueOf(key), pt)
		slice = reflect.Append(slice, v)
		return nil
	})

	p.Set(slice)
}

func mapValues(m Map) []interface{} {
	values := make([]interface{}, m.Size())

	i := 0
	m.ForEach(func(key, value interface{}) error {
		values[i] = value
		i++

		return nil
	})

	return values
}

func mapValuesInto(m Map, into interface{}) {
	p := reflect.ValueOf(into).Elem()
	pt := p.Type().Elem()

	slice := p

	m.ForEach(func(key, value interface{}) error {
		v := coalesceInvalidToZeroValueOf(reflect.ValueOf(value), pt)
		slice = reflect.Append(slice, v)
		return nil
	})

	p.Set(slice)
}

func mapForEachInto(m Map, fn interface{}) error {
	fnr := reflect.ValueOf(fn)
	fnt := fnr.Type()

	if fnt.NumOut() != 1 {
		panic(ErrInvalidFuncSignature)
	}

	return m.ForEach(func(key, value interface{}) error {
		p1 := coalesceInvalidToZeroValueOf(reflect.ValueOf(key), fnt.In(0))
		p2 := coalesceInvalidToZeroValueOf(reflect.ValueOf(value), fnt.In(1))
		r := fnr.Call([]reflect.Value{p1, p2})

		err := r[0]
		if err.IsNil() {
			return nil
		}

		return err.Interface().(error)
	})
}
