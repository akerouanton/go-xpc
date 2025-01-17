package xpc

/*
#include <xpc/xpc.h>
#cgo CFLAGS: -x objective-c
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

func Marshal(v any) (C.xpc_object_t, error) {
	st := reflect.TypeOf(v)
	if st == nil {
		return nil, nil
	}

	encoded, err := marshalVal(v)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func marshalVal(val any) (C.xpc_object_t, error) {
	// Check if val implements error interface
	if err, ok := val.(error); ok {
		// If it's a struct or has fields we want to preserve, marshal it as a dictionary
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Struct || (v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct) {
			dict := C.xpc_dictionary_create_empty()
			C.xpc_dictionary_set_value(dict, C.CString("_error"), C.xpc_string_create(C.CString(err.Error())))

			elemVal := val
			if v.Kind() == reflect.Ptr {
				elemVal = v.Elem().Interface()
			}
			// Marshal the rest of the struct fields
			if err := marshalIntoDict(dict, elemVal); err != nil {
				return nil, err
			}
			return dict, nil
		}
		// For simple errors, just marshal the error message as a string
		return C.xpc_string_create(C.CString(err.Error())), nil
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Bool:
		return C.xpc_bool_create(C.bool(v.Bool())), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return C.xpc_int64_create(C.int64_t(v.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return C.xpc_uint64_create(C.uint64_t(v.Uint())), nil
	case reflect.Float32, reflect.Float64:
		return C.xpc_double_create(C.double(v.Float())), nil
	case reflect.String:
		return C.xpc_string_create(C.CString(v.String())), nil
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		arr := C.xpc_array_create_empty()
		for i := 0; i < v.Len(); i++ {
			item, err := Marshal(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			C.xpc_array_append_value(arr, item)
		}
		return arr, nil
	case reflect.Struct:
		dict := C.xpc_dictionary_create_empty()
		if err := marshalIntoDict(dict, val); err != nil {
			return nil, err
		}
		return dict, nil
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}
		return Marshal(v.Elem().Interface())
	default:
		return nil, errors.New("unsupported type: " + v.Kind().String())
	}
}

func marshalIntoDict(dst C.xpc_object_t, src any) error {
	v := reflect.ValueOf(src)
	vtype := v.Type()
	for i := 0; i < vtype.NumField(); i++ {
		field := vtype.Field(i)
		if !field.IsExported() {
			continue
		}

		xpcKey := field.Tag.Get("xpc")
		if xpcKey == "" {
			xpcKey = field.Name
		}
		if strings.HasPrefix(xpcKey, "_") {
			return fmt.Errorf("xpc key cannot start with underscore: %s", xpcKey)
		}

		item, err := Marshal(v.Field(i).Interface())
		if err != nil {
			return err
		}
		C.xpc_dictionary_set_value(dst, C.CString(xpcKey), item)
	}
	return nil
}

func getXPCType(v interface{}) string {
	vv, ok := v.(C.xpc_object_t)
	if !ok {
		return ""
	}
	return C.GoString(C.xpc_type_get_name(C.xpc_get_type(vv)))
}

func Unmarshal(msg unsafe.Pointer, v interface{}) error {
	if msg == nil {
		return nil
	}

	obj := (C.xpc_object_t)(msg)

	// Get the reflect value we're unmarshaling into
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("unmarshal target must be a non-nil pointer")
	}

	// Get the element the pointer points to
	rv = rv.Elem()
	result, err := unmarshalVal(obj, rv.Type())
	if err != nil {
		return err
	}

	setVal(rv, result)
	return nil
}

func setVal(rv reflect.Value, val interface{}) {
	switch rv.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		// For primitive types, set directly
		rv.Set(reflect.ValueOf(val))
	case reflect.Ptr:
		// For pointers, we need to create a new pointer value and set it to point to the result
		newPtr := reflect.New(reflect.ValueOf(val).Type())
		newPtr.Elem().Set(reflect.ValueOf(val))
		rv.Set(newPtr)
	case reflect.Slice:
		// For slices, we need to create a new slice of the correct type
		if resultSlice, ok := val.([]interface{}); ok {
			newSlice := reflect.MakeSlice(rv.Type(), len(resultSlice), cap(resultSlice))
			for i, val := range resultSlice {
				newSlice.Index(i).Set(reflect.ValueOf(val))
			}
			rv.Set(newSlice)
		} else {
			rv.Set(reflect.ValueOf(val))
		}
	default:
		// Try direct setting for other types
		rv.Set(reflect.ValueOf(val))
	}
}

func unmarshalVal(obj C.xpc_object_t, typ reflect.Type) (interface{}, error) {
	switch C.xpc_get_type(obj) {
	case C.XPC_TYPE_BOOL:
		return bool(C.xpc_bool_get_value(obj)), nil

	case C.XPC_TYPE_INT64:
		val := int64(C.xpc_int64_get_value(obj))
		// Convert to the target type if needed
		switch typ.Kind() {
		case reflect.Int:
			return int(val), nil
		case reflect.Int8:
			return int8(val), nil
		case reflect.Int16:
			return int16(val), nil
		case reflect.Int32:
			return int32(val), nil
		default:
			return val, nil
		}

	case C.XPC_TYPE_UINT64:
		val := uint64(C.xpc_uint64_get_value(obj))
		// Convert to the target type if needed
		switch typ.Kind() {
		case reflect.Uint:
			return uint(val), nil
		case reflect.Uint8:
			return uint8(val), nil
		case reflect.Uint16:
			return uint16(val), nil
		case reflect.Uint32:
			return uint32(val), nil
		default:
			return val, nil
		}

	case C.XPC_TYPE_DOUBLE:
		val := float64(C.xpc_double_get_value(obj))
		if typ.Kind() == reflect.Float32 {
			return float32(val), nil
		}
		return val, nil

	case C.XPC_TYPE_STRING:
		return C.GoString(C.xpc_string_get_string_ptr(obj)), nil

	case C.XPC_TYPE_ARRAY:
		count := int(C.xpc_array_get_count(obj))
		if typ.Kind() == reflect.Array {
			// Fixed size array
			result := reflect.New(typ).Elem()
			for i := 0; i < count && i < typ.Len(); i++ {
				item := C.xpc_array_get_value(obj, C.size_t(i))
				val, err := unmarshalVal(item, typ.Elem())
				if err != nil {
					return nil, err
				}
				result.Index(i).Set(reflect.ValueOf(val))
			}
			return result.Interface(), nil
		} else {
			// Slice
			result := make([]interface{}, count)
			for i := 0; i < count; i++ {
				item := C.xpc_array_get_value(obj, C.size_t(i))
				val, err := unmarshalVal(item, typ.Elem())
				if err != nil {
					return nil, err
				}
				result[i] = val
			}
			return result, nil
		}

	case C.XPC_TYPE_DICTIONARY:
		if typ.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			// If it's an error type, unmarshal as a string
			item := C.xpc_dictionary_get_value(obj, C.CString("_error"))
			if item != nil {
				errStr := C.GoString(C.xpc_string_get_string_ptr(item))
				return errors.New(errStr), nil
			}
			return nil, nil
		}
		if typ.Kind() == reflect.Struct {
			result := reflect.New(typ).Elem()
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				key := field.Tag.Get("xpc")
				if key == "" {
					key = field.Name
				}
				ckey := C.CString(key)
				defer C.free(unsafe.Pointer(ckey))

				item := C.xpc_dictionary_get_value(obj, ckey)
				if item != nil {
					val, err := unmarshalVal(item, field.Type)
					if err != nil {
						return nil, err
					}
					setVal(result.Field(i), val)
				}
			}
			return result.Interface(), nil
		} else {
			return nil, errors.New("unmarshal target must be a struct, maps not supported")
		}

	default:
		return nil, errors.New("unsupported XPC type")
	}
}
