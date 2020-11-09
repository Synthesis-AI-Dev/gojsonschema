package gojsonschema

import (
	"encoding/json"
	"log"
	"reflect"
	"testing"
)

func M(in ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if len(in)%2 != 0 {
		log.Fatal("map construction M must have one value for each key")
	}

	for i := 0; i < len(in); i += 2 {
		k := in[i]
		v := in[i+1]
		sK := k.(string)
		result[sK] = v
	}

	return result
}

// objSchemaFromProperties constructs a schema from "properties"
func objSchemaFromProperties(properties map[string]interface{}) *Schema {
	schemaMap := M("type", "object", "properties", properties)
	loader := NewGoLoader(schemaMap)

	// Since its a test, it'll just fail and then we address later
	schema, _ := NewSchema(loader)

	return schema
}

// arrSchemaFromProperties makes sure that each item in an array contains
// the properties passed in "properties"
func arrSchemaFromProperties(properties map[string]interface{}) *Schema {
	objMap := M("type", "object", "properties", properties)
	arrMap := M("type", "array", "items", objMap)

	loader := NewGoLoader(arrMap)

	// Since its a test, it'll just fail and then we address later
	schema, _ := NewSchema(loader)

	return schema
}

// InsertDefaults fails when nil is passed as argument
func TestInsertNil(t *testing.T) {
	properties := M()
	schema := objSchemaFromProperties(properties)

	_, err := schema.InsertDefaults(nil)
	if err == nil {
		t.Error("InsertDefault should fail with a nil argument")
	}
}

// InsertDefaults succeeds when simple map is passed as argument
func TestSimpleDefault(t *testing.T) {
	properties := M("testkey", M("default", "defaultvalue"))
	schema := objSchemaFromProperties(properties)

	into := make(map[string]interface{})

	r, err := schema.InsertDefaults(into)
	if err != nil {
		t.Error(err)
	}

	result := r.(map[string]interface{})

	if v := result["testkey"]; v != "defaultvalue" {
		t.Error("InsertDefaults failed to add 'defaultvalue' at 'testkey'")
	}
}

func TestDoesNotOverwrite(t *testing.T) {
	properties := M("testkey", M("default", "defaultvalue"))
	schema := objSchemaFromProperties(properties)

	into := make(map[string]interface{})
	into["testkey"] = "someothervalue"

	r, err := schema.InsertDefaults(into)
	if err != nil {
		t.Error(err)
	}

	result := r.(map[string]interface{})

	if v := result["testkey"]; v != "someothervalue" {
		t.Error("InsertDefaults has overwritten a value that was there before")
	}
}

// TestNestedValues makes sure that a default value several layers deep will be inserted
func TestNestedValues(t *testing.T) {
	properties := M("deep", M("type", "object", "properties", M("testkey", M("default", "defaultvalue"))))
	schema := objSchemaFromProperties(properties)

	into := make(map[string]interface{})

	r, err := schema.InsertDefaults(into)
	if err != nil {
		t.Error(err)
	}

	result := r.(map[string]interface{})

	innerMap := result["deep"].(map[string]interface{})

	if v := innerMap["testkey"]; v != "defaultvalue" {
		t.Error("InsertDefaults failed to add 'defaultvalue' at .'deep'.'testkey'")
	}
}

// If an empty array is passed, nothing should be inserted, even if there
// is a default value specified somewhere
func TestSimpleArr(t *testing.T) {
	properties := M("testkey", M("default", "defaultvalue"))
	schema := arrSchemaFromProperties(properties)

	examplearr := make([]map[string]interface{}, 0)

	_, err := schema.InsertDefaults(examplearr)
	if err != nil {
		t.Error(err)
	}
}

func TestArrayOfProperties(t *testing.T) {
	properties := M("testkey", M("default", "defaultvalue"))
	schema := arrSchemaFromProperties(properties)

	emptyexample := M()

	examplearr := make([]map[string]interface{}, 1)
	examplearr[0] = emptyexample

	r, err := schema.InsertDefaults(examplearr)
	if err != nil {
		t.Error(err)
	}

	if res := r.([]map[string]interface{}); res[0]["testkey"] != "defaultvalue" {
		t.Error("array[0] was not filled with the proper default values")
	}
}

// TestTypeObjectOrNull make sure that a type that is an array containing
// "object" will succeed
func TestTypeObjectOrNull(t *testing.T) {
	tdef := []string{
		"object",
		"null",
	}
	properties := M("deep", M("type", tdef, "properties", M("testkey", M("default", "defaultvalue"))))
	schema := objSchemaFromProperties(properties)

	into := make(map[string]interface{})

	r, err := schema.InsertDefaults(into)
	if err != nil {
		t.Error(err)
	}

	result := r.(map[string]interface{})

	innerMap := result["deep"].(map[string]interface{})

	if v := innerMap["testkey"]; v != "defaultvalue" {
		t.Error("InsertDefaults failed to add 'defaultvalue' at .'deep'.'testkey'")
	}
}

// TestTypeArrayOrNull make sure that a type that is an array containing
// "object" will succeed
func TestTypeArrayOrNull(t *testing.T) {
	properties := M("testkey", M("default", "defaultvalue"))

	tdef := []string{
		"array",
		"null",
	}
	objMap := M("type", "object", "properties", properties)
	arrMap := M("type", tdef, "items", objMap)

	loader := NewGoLoader(arrMap)
	schema, _ := NewSchema(loader)

	emptyexample := M()

	examplearr := make([]map[string]interface{}, 1)
	examplearr[0] = emptyexample

	r, err := schema.InsertDefaults(examplearr)
	if err != nil {
		t.Error(err)
	}

	if res := r.([]map[string]interface{}); res[0]["testkey"] != "defaultvalue" {
		t.Error("array[0] was not filled with the proper default values")
	}
}

// TestDefaultsWithinArray make sure that if an array is null but
// a default value is set for the array that the default value will be
// set.
func TestDefaultsWithinArray(t *testing.T) {
	fProps := M("hdri", M("type", "string", "default", "apartment"))

	face := M("type", "object", "properties", fProps)

	fArray := M("type", "array", "items", face)
	faces := M("faces", fArray)

	req := M("type", "object", "properties", faces)

	// object with array of objects
	// we want to test properties of the inner array
	// {
	// 	"type": "object",
	// 	"properties": {
	// 		"faces": {
	// 			"type": "array",
	// 			"items": {
	// 				"type": "object",
	// 				"properties": {
	// 					"hdri": {
	// 						"type": "string",
	// 						"default": "apartment"
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	loader := NewGoLoader(req)
	schema, _ := NewSchema(loader)

	exampleFaces := make([]map[string]interface{}, 1)
	exampleFaces[0] = make(map[string]interface{})

	exampleReq := make(map[string]interface{})
	exampleReq["faces"] = exampleFaces

	r, err := schema.InsertDefaults(exampleReq)
	if err != nil {
		t.Error(err)
	}

	r0, ok := r.(map[string]interface{})
	if ok == false {
		t.Error("err casting original request to map[string]interface{}")
	}

	r1, ok := r0["faces"].([]map[string]interface{})
	if ok == false {
		t.Error("err casting faces to []map[string]interface{}")
	}

	f := r1[0]
	if f["hdri"] != "apartment" {
		t.Error("default value was not set in the 0th array item")
	}
}

// TestInputFromString makes sure generics encountered when interpreting
// the schema are handled properly
func TestInputFromString(t *testing.T) {
	fProps := M("hdri", M("type", "string", "default", "apartment"))

	face := M("type", "object", "properties", fProps)

	fArray := M("type", "array", "items", face)
	faces := M("faces", fArray)

	req := M("type", "object", "properties", faces)

	// object with array of objects
	// we want to test properties of the inner array
	// {
	// 	"type": "object",
	// 	"properties": {
	// 		"faces": {
	// 			"type": "array",
	// 			"items": {
	// 				"type": "object",
	// 				"properties": {
	// 					"hdri": {
	// 						"type": "string",
	// 						"default": "apartment"
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	loader := NewGoLoader(req)
	schema, _ := NewSchema(loader)

	// unmarshal string to request
	exReq := `{"faces":[{"lskdjf":"sdf"}]}`
	i := make(map[string]interface{}, 100)
	if err := json.Unmarshal([]byte(exReq), &i); err != nil {
		t.Error("failed to unmarshal request into byte slice")
	}

	r, err := schema.InsertDefaults(i)
	if err != nil {
		t.Error(err)
	}

	r0, ok := r.(map[string]interface{})
	if ok == false {
		t.Error("err casting original request to map[string]interface{}")
	}

	fs := r0["faces"]
	vr := reflect.ValueOf(fs)
	nv := make([]map[string]interface{}, vr.Len())
	for i := 0; i < vr.Len(); i++ {
		tmp := vr.Index(i).Interface()
		nv[i] = tmp.(map[string]interface{})
	}

	f := nv[0]
	if f["hdri"] != "apartment" {
		t.Error("default value was not set in the 0th array item")
	}
}

// TestOneOf make sure the default library applies a default value to
// cases where oneOf is used.
func TestOneOf(t *testing.T) {
	d := M("hello", "world")

	// note the parameter passed as "" doesn't matter here because
	// we never actually look at what oneOf is
	nm := M("type", "object", "oneOf", "", "default", d)
	name := M("name", nm)

	req := M("type", "object", "properties", name)

	// object with array of objects
	// we want to test properties of the inner array
	// {
	// 		"type": "object",
	// 		"properties": {
	// 			"name": {
	//				"type": "object"
	//				"oneOf": <doesn't matter for this test>,
	// 				"default": { "hello": "world" }
	//			}
	// 		}
	// }
	loader := NewGoLoader(req)
	schema, _ := NewSchema(loader)

	// unmarshal string to request
	exReq := `{}`
	i := make(map[string]interface{}, 100)
	if err := json.Unmarshal([]byte(exReq), &i); err != nil {
		t.Error("failed to unmarshal request into byte slice")
	}

	r, err := schema.InsertDefaults(i)
	if err != nil {
		t.Error(err)
	}

	r0, ok := r.(map[string]interface{})
	if ok == false {
		t.Error("err casting original request to map[string]interface{}")
	}

	fs := r0["faces"]
	vr := reflect.ValueOf(fs)
	nv := make([]map[string]interface{}, vr.Len())
	for i := 0; i < vr.Len(); i++ {
		tmp := vr.Index(i).Interface()
		nv[i] = tmp.(map[string]interface{})
	}

	f := nv[0]
	if f["hdri"] != "apartment" {
		t.Error("default value was not set in the 0th array item")
	}
}
