// Package keepstate turns a program's values into JSON and back, unexported
// fields and all, so that an app comes back as it was after a restart, without
// tags or methods on its types: a struct of its own, as main packages keep theirs,
// just works.
//
// It keeps data, not the world around it: what a function, a channel, an interface
// or a pointer to another package's type holds stays as the program made it. A
// field whose type changed since the value was saved, as a change of the code may
// have done, stays as it is too.
package keepstate

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// maxDepth bounds what a cycle of pointers can do.
const maxDepth = 64

var timeType = reflect.TypeFor[time.Time]()

// Encode returns v, a pointer, as JSON. Values it can't walk it leaves out.
func Encode(v any) ([]byte, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Pointer || r.IsNil() {
		return nil, nil
	}
	return json.Marshal(encode(r.Elem(), pkgOf(r.Elem().Type()), 0))
}

// Decode sets what data holds in v, a pointer, as far as the two still agree.
func Decode(data []byte, v any) error {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Pointer || r.IsNil() {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // integers whole
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return err
	}
	decode(r.Elem(), tree, pkgOf(r.Elem().Type()), 0)
	return nil
}

// pkgOf returns the package a value belongs to: of its element, for a slice or a
// map of its structs.
func pkgOf(t reflect.Type) string {
	for {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		default:
			return t.PkgPath()
		}
	}
}

// walkable says whether to go into a struct: not the standard library's, whose
// locks and buffers restart as they are.
func walkable(t reflect.Type) bool {
	path := t.PkgPath()
	first, _, _ := strings.Cut(path, "/")
	return path == "main" || strings.Contains(first, ".")
}

// followed says whether to go through a pointer to t: to data of the value's own
// package, not to the windows, cameras and bitmaps of others.
func followed(t reflect.Type, pkg string) bool {
	switch t.Kind() {
	case reflect.Struct:
		return t.PkgPath() == pkg
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Interface:
		return false
	}
	return true
}

func encode(v reflect.Value, pkg string, depth int) any {
	if depth > maxDepth {
		return nil
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if f != f || f > 1.7e308 || f < -1.7e308 {
			return nil // JSON has none of those
		}
		return f
	case reflect.String:
		return v.String()
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		out := make([]any, v.Len())
		for i := range out {
			out[i] = encode(v.Index(i), pkg, depth+1)
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := make(map[string]any, v.Len())
		for it := v.MapRange(); it.Next(); {
			if k, ok := keyString(it.Key()); ok {
				out[k] = encode(it.Value(), pkg, depth+1)
			}
		}
		return out
	case reflect.Pointer:
		if v.IsNil() || !followed(v.Type().Elem(), pkg) {
			return nil
		}
		return encode(v.Elem(), pkg, depth+1)
	case reflect.Struct:
		if v.Type() == timeType {
			t := reachable(v).Interface().(time.Time)
			if b, err := t.MarshalText(); err == nil {
				return string(b)
			}
			return nil
		}
		if !walkable(v.Type()) {
			return nil
		}
		out := make(map[string]any, v.NumField())
		t := v.Type()
		for i := range t.NumField() {
			if f := encode(v.Field(i), pkg, depth+1); f != nil {
				out[t.Field(i).Name] = f
			}
		}
		return out
	}
	return nil
}

func keyString(k reflect.Value) (string, bool) {
	switch k.Kind() {
	case reflect.String:
		return k.String(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(k.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(k.Uint(), 10), true
	}
	return "", false
}

func keyValue(t reflect.Type, s string) (reflect.Value, bool) {
	k := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.String:
		k.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, t.Bits())
		if err != nil {
			return k, false
		}
		k.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, t.Bits())
		if err != nil {
			return k, false
		}
		k.SetUint(n)
	default:
		return k, false
	}
	return k, true
}

// reachable returns v without the read-only mark of an unexported field.
func reachable(v reflect.Value) reflect.Value {
	if v.CanSet() || !v.CanAddr() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func decode(v reflect.Value, tree any, pkg string, depth int) {
	if tree == nil || depth > maxDepth {
		return // saved as nothing, or left out: what the program made stands
	}
	v = reachable(v)
	if !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		if b, ok := tree.(bool); ok {
			v.SetBool(b)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := tree.(json.Number); ok {
			if i, err := n.Int64(); err == nil && !v.OverflowInt(i) {
				v.SetInt(i)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if n, ok := tree.(json.Number); ok {
			if i, err := strconv.ParseUint(n.String(), 10, 64); err == nil && !v.OverflowUint(i) {
				v.SetUint(i)
			}
		}
	case reflect.Float32, reflect.Float64:
		if n, ok := tree.(json.Number); ok {
			if f, err := n.Float64(); err == nil && !v.OverflowFloat(f) {
				v.SetFloat(f)
			}
		}
	case reflect.String:
		if s, ok := tree.(string); ok {
			v.SetString(s)
		}
	case reflect.Slice:
		if list, ok := tree.([]any); ok {
			out := reflect.MakeSlice(v.Type(), len(list), len(list))
			for i, x := range list {
				decode(out.Index(i), x, pkg, depth+1)
			}
			v.Set(out)
		}
	case reflect.Array:
		if list, ok := tree.([]any); ok {
			for i, x := range list {
				if i < v.Len() {
					decode(v.Index(i), x, pkg, depth+1)
				}
			}
		}
	case reflect.Map:
		if m, ok := tree.(map[string]any); ok {
			out := reflect.MakeMapWithSize(v.Type(), len(m))
			for s, x := range m {
				k, ok := keyValue(v.Type().Key(), s)
				if !ok {
					continue
				}
				e := reflect.New(v.Type().Elem()).Elem()
				decode(e, x, pkg, depth+1)
				out.SetMapIndex(k, e)
			}
			v.Set(out)
		}
	case reflect.Pointer:
		if !followed(v.Type().Elem(), pkg) {
			return
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		decode(v.Elem(), tree, pkg, depth+1)
	case reflect.Struct:
		if v.Type() == timeType {
			if s, ok := tree.(string); ok {
				var t time.Time
				if t.UnmarshalText([]byte(s)) == nil {
					v.Set(reflect.ValueOf(t))
				}
			}
			return
		}
		m, ok := tree.(map[string]any)
		if !ok || !walkable(v.Type()) {
			return
		}
		t := v.Type()
		for i := range t.NumField() {
			if x, ok := m[t.Field(i).Name]; ok {
				decode(v.Field(i), x, pkg, depth+1)
			}
		}
	}
}
