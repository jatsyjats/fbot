// Common functionality and utilities

package main

import (
	"fmt"
	"github.com/go-errors/errors"
	"log"
	"runtime"
	"strings"
)

type Error = *errors.Error

func Logf(format string, args ...any) {
	pc, file, line, _ := runtime.Caller(1)

	files := strings.Split(file, "/")
	file = files[len(files)-1]

	name := runtime.FuncForPC(pc).Name()
	fns := strings.Split(name, ".")
	name = fns[len(fns)-1]

	msg := fmt.Sprintf(format, args...)

	log.Printf("[FBot] %s:%d:%s() %s\n", file, line, name, msg)
}

func WrapError(err error) Error {
	return errors.Wrap(err, 1)
}

func ErrorToStr(err error) string {
	errWithStack, ok := err.(*errors.Error)
	if ok {
		return errWithStack.ErrorStack()
	} else {
		return err.Error()
	}
}
