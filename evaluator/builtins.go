package evaluator

import (
	"bufio"
	"fmt"
	"monkey/object"
	"os"
	"strings"
)

//
// Global STDIN-reader.
//
var reader = bufio.NewReader(os.Stdin)

//
// Mapping of file-IDs to file-handles
//
var file_handles = make(map[uintptr]*os.File)
var file_readers = make(map[uintptr]*bufio.Reader)

//
// Horrid hack - setup STDIN/STDOUT/STDERR
//
func setupHandles() {
	if file_handles[0] != nil {
		fmt.Printf("Already setup..")
		return
	}
	file_handles[0] = os.Stdin
	file_handles[1] = os.Stdout
	file_handles[2] = os.Stderr

	file_readers[0] = bufio.NewReader(os.Stdin)
	file_readers[1] = bufio.NewReader(os.Stdout)
	file_readers[2] = bufio.NewReader(os.Stderr)
}

// builtin function maps
var builtins = map[string]*object.Builtin{
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("argument to `len` not supported, got=%s",
					args[0].Type())
			}
		},
	},
	"first": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `first` must be ARRAY, got %s",
					args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				return arr.Elements[0]
			}
			return NULL
		},
	},
	"last": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `last` must be ARRAY, got %s",
					args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				return arr.Elements[length-1]
			}
			return NULL
		},
	},
	"rest": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `rest` must be ARRAY, got=%s",
					args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				newElements := make([]object.Object, length-1, length-1)
				copy(newElements, arr.Elements[1:length])
				return &object.Array{Elements: newElements}
			}
			return NULL

		},
	},
	"push": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `push` must be ARRAY, got=%s",
					args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			newElements := make([]object.Object, length+1, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]
			return &object.Array{Elements: newElements}
		},
	},
	"set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("wrong number of arguments. got=%d, want=2",
					len(args))
			}
			if args[0].Type() != object.HASH_OBJ {
				return newError("argument to `set` must be HASH, got=%s",
					args[0].Type())
			}
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("key `set` into HASH must be Hashable, got=%s",
					args[1].Type())
			}
			newHash := make(map[object.HashKey]object.HashPair)
			hash := args[0].(*object.Hash)
			for k, v := range hash.Pairs {
				newHash[k] = v
			}
			newHashKey := key.HashKey()
			newHashPair := object.HashPair{Key: args[1], Value: args[2]}
			newHash[newHashKey] = newHashPair
			return &object.Hash{Pairs: newHash}
		},
	},
	"puts": {
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Print(arg.Inspect())
			}
			return NULL
		},
	},
	"type": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			switch args[0].(type) {
			case *object.String:
				return &object.String{Value: "string"}
			case *object.Boolean:
				return &object.String{Value: "bool"}
			case *object.Array:
				return &object.String{Value: "array"}
			case *object.Function:
				return &object.String{Value: "function"}
			case *object.Integer:
				return &object.String{Value: "integer"}
			case *object.Float:
				return &object.String{Value: "float"}
			case *object.Hash:
				return &object.String{Value: "hash"}
			default:
				return newError("argument to `type` not supported, got=%s",
					args[0].Type())
			}
		},
	},
	"read": {
		Fn: func(args ...object.Object) object.Object {
			setupHandles()
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			id := args[0].(*object.Integer).Value
			reader := file_readers[uintptr(id)]

			line, err := reader.ReadString('\n')
			if err == nil {
				return &object.String{Value: line}
			} else {
				return &object.String{Value: ""}
			}
		},
	},
	"split": {
		Fn: func(args ...object.Object) object.Object {
			//
			// Default separator.
			//
			sep := " "

			if len(args) != 1 && len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=1 or 2.",
					len(args))
			}
			if args[0].Type() != object.STRING_OBJ {
				return newError("argument to `split` must be a string, got=%s",
					args[0].Type())
			}
			if len(args) == 2 {
				if args[1].Type() != object.STRING_OBJ {
					return newError("argument to `split` must be a string, got=%s",
						args[0].Type())
				}
				sep = args[1].(*object.String).Value

			}

			// split by separator
			fields := strings.Split(args[0].(*object.String).Value, sep)

			// make results
			l := len(fields)
			result := make([]object.Object, l, l)
			for i, txt := range fields {
				result[i] = &object.String{Value: txt}
			}
			return &object.Array{Elements: result}
		},
	},
	"fopen": {
		Fn: func(args ...object.Object) object.Object {
			setupHandles()
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			path := args[0].(*object.String).Value
			file, err := os.Open(path)
			if err != nil {
				return &object.Integer{Value: -1}
			}

			// convert handle to integer to return it
			file_handles[file.Fd()] = file
			// but also store a reader
			file_readers[file.Fd()] = bufio.NewReader(file)

			return &object.Integer{Value: int64(file.Fd())}
		},
	},
	"fclose": {
		Fn: func(args ...object.Object) object.Object {
			setupHandles()
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			handle := args[0].(*object.Integer).Value

			file_handles[uintptr(handle)].Close()
			delete(file_handles, uintptr(handle))
			delete(file_readers, uintptr(handle))
			return NULL
		},
	},
}
