package main

import (
	"fmt"
	"os"
)

/* Module for error handling and such. */

// Functional error handling when function returns an argument.
// Note that the parameters are reversed: this is because the type cannot be inferred otherwise.
//
//	The user must now make sure that the function after this is called.
func failIf[T any](data T, err error) func(errmsg string) T {
	return func(errmsg string) T {
		if err != nil {
			fail(errmsg+":", err)
		}

		return data
	}
}

// Similar thing like `failWhen`, but without arguments.
func failWhen(errmsg string) func(err error) {
	return func(err error) {
		if err != nil {
			fail(errmsg+":", err)
		}
	}
}

// this is getting funny
func failCleanupWhen(errmsg string) func(func()) func(error) {
	return func(f func()) func(error) {
		return func(err error) {
			if err != nil {
				f()
				fail(errmsg+":", err)
			}
		}
	}
}

func fail(err ...any) {
	fmt.Println(err...)
	os.Exit(1)
}
