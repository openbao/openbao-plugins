package datastructure

import (
	"reflect"
)

func setValuesInto(s Set, into interface{}) {
	p := reflect.ValueOf(into).Elem()
	pt := p.Type().Elem()

	slice := p

	s.ForEach(func(element interface{}) error {
		v := coalesceInvalidToZeroValueOf(reflect.ValueOf(element), pt)
		slice = reflect.Append(slice, v)
		return nil
	})

	p.Set(slice)
}

func setForEachInto(s Set, fn interface{}) error {
	fnr := reflect.ValueOf(fn)
	fnt := fnr.Type()

	if fnt.NumOut() != 1 {
		panic(ErrInvalidFuncSignature)
	}

	return s.ForEach(func(element interface{}) error {
		p := coalesceInvalidToZeroValueOf(reflect.ValueOf(element), fnt.In(0))
		r := fnr.Call([]reflect.Value{p})

		err := r[0]
		if err.IsNil() {
			return nil
		}

		return err.Interface().(error)
	})
}
