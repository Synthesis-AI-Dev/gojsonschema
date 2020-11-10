package gojsonschema

import (
	"errors"
	"fmt"
	"reflect"
)

// Gotchas and todos:
// - a schema can contain multiple types, provided that only one is non-null,
//   for example, both "array", and ["array", "null"] are valid
// - oneOf is not supported, but a default value can be set to handle cases where
//   the attribute isn't defined at all
// - does not currently support array of arrays
//
//
// This code was inspired by, and follows the same basic structure as
// https://github.com/juju/gojsonschema. The main notable differences are:
//
// 1. s.pool.GetStandaloneDocument is no longer a method in gojsonschema,
//    and document interface must be retrieved by pulling from document pool
// 2. Array values were not supported in juju/gojsonschema
//
// Tests in this repo are distinct from those in the above repo, but do copy
// the method `func M(in ...interface{}) map[string]interface{}`, which
// I found useful for quickly constructing arbitrary maps

// InsertDefaults takes a generic interface (because it could either be an
// object or an array, and attemps to fill it with as many defaults as possible
func (s *Schema) InsertDefaults(into interface{}) (m interface{}, returnErr error) {
	defer panicHandler(&returnErr)

	// We need to get the outermost document before entering the recursive function
	// because we'll recurse down into this map as well.
	schemaMap, err := s.getDocumentMap()
	if err != nil {
		return nil, err
	}

	insertRecursively(into, schemaMap)

	return into, nil // err is filled if panic
}

func panicHandler(err *error) {
	if r := recover(); r != nil {
		var msg string
		switch t := r.(type) {
		case error:
			msg = fmt.Sprintf("schema error caused a panic: %s", t.Error())
		default:
			msg = fmt.Sprintf("unknown panic: %#v", t)
		}
		*err = errors.New(msg)
	}
}

func (s *Schema) getDocumentMap() (m map[string]interface{}, err error) {
	f, err := s.pool.GetDocument(s.documentReference)
	if err != nil {
		return nil, err
	}
	return f.Document.(map[string]interface{}), nil
}

// insertRecursively inserts into "into", which is either an array or an object
func insertRecursively(into interface{}, from map[string]interface{}) {
	if v, ok := from["type"]; ok {
		switch t := *typeIgnoreNull(v); t {

		case "array":
			// intoAsArray represents many objects of the same type
			intoAsArray := into.([]map[string]interface{})

			nextMap := from["items"].(map[string]interface{})
			for _, example := range intoAsArray {
				insertRecursively(example, nextMap)
			}

		case "object":
			intoAsObject := into.(map[string]interface{})

			// an object with oneOf doesn't have a "properties" attribute, so
			// return from this instead of trying to iterate them
			if _, ok := from["oneOf"]; ok {
				return
			}

			// nextMap represents the subSchema that we want this single item to
			// conform to
			properties := from["properties"].(map[string]interface{})

			for property, _nextSchema := range properties {
				nextSchema := _nextSchema.(map[string]interface{})

				// This block becomes active if value already exists in "into"
				// for this property and value is not null.
				//
				// If v is null, we want to progress to the next block in case
				// a default value exists and should be applied on top.
				if v, ok := intoAsObject[property]; ok && v != nil {
					vr := reflect.ValueOf(v)
					switch vr.Kind() {

					case reflect.Map:
						if innerMapAsObj, ok := v.(map[string]interface{}); ok {
							insertRecursively(innerMapAsObj, nextSchema)
						}

					// object.properties is always going to be map[string]interface{} because
					// we do not current support arrays of arrays
					case reflect.Slice:
						arrSchema := nextSchema["items"].(map[string]interface{})
						arrType := *typeIgnoreNull(arrSchema["type"])

						// If "into" has stuff in it, and its an array, we need to figure out if the stuff is
						// an object before we decide to recurse further. Otherwise, defaults are irrelevant
						// and we stop
						if arrType == "object" {
							// was having trouble casting directly to []map[string]interface{}.
							// a, ok := v.([]map[string]interface{}) was setting a => false. Did a
							// bit of digging and found https://stackoverflow.com/a/12754757/1461022.
							// It seems like go wants me to loop through the items and cast them explicitly
							nv := make([]map[string]interface{}, vr.Len())
							for i := 0; i < vr.Len(); i++ {
								tmp := vr.Index(i).Interface()
								nv[i] = tmp.(map[string]interface{})
							}
							insertRecursively(nv, nextSchema)
						}
					}
					continue
				}

				// We can't step deeper so we're at an actual key/value
				// Check to see if we should add a default
				if d, ok := nextSchema["default"]; ok {
					intoAsObject[property] = d
					continue
				}

				// Finally, if the next schema does exists but there is nothing
				// in the input object, we want to create a temporary interface, just
				// in case a nested object has defaults
				//
				// This is one case where the specifications do not provide details around
				// how default values must play with other properties, for example, if
				// the user provides "null" in a given request (which is valid) and a
				// default value exists in the inner object specs (which is also valid), we
				// set the default value, upgrading the interpreted type from "null" to "object"
				if v, ok := nextSchema["type"]; ok {
					switch *typeIgnoreNull(v) {
					case "object":
						tmpTarget := make(map[string]interface{})
						insertRecursively(tmpTarget, nextSchema)
						if len(tmpTarget) > 0 {
							intoAsObject[property] = tmpTarget
						}
					}
				}
			}
		}
	}
}

// stringExistsInSlice looks for str in slice. At the
// moment, []interface{} is used because go doesn't know that
// types must be a string, hence the compiler does not like that
// I'm trying to slice as strictly containing them
func stringExistsInSlice(slice []interface{}, val string) bool {
	for _, item := range slice {
		s := fmt.Sprintf("%v", item)
		if s == val {
			return true
		}
	}
	return false
}

// return "string" if one of the following are matched
// 1. type: "string"
// 2. type: ["string", "null"]
func typeIgnoreNull(v interface{}) *string {
	switch v.(type) {
	case string:
		s := v.(string)
		return &s
	case []interface{}:
		// according to jsonschema this must be a string, but go doesn't
		// know that, and for all it knows this could be any type
		s := v.([]interface{})
		o := "object"
		a := "array"

		if stringExistsInSlice(s, o) {
			return &o
		}
		if stringExistsInSlice(s, a) {
			return &a
		}
	}

	return nil
}
