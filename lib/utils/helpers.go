package helpers

import (
	"path/filepath"
	"runtime"
	"strings"
)

func GetCallerInfo(level int) (string, string, int) {
	pc, fileName, line, ok := runtime.Caller(level)
	if !ok {
		return "_", "_", 0
	}
	fileName = filepath.Base(fileName)
	// Get the name of the function from the program counter (pc)
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "_", "_", 0
	}
	// Extract and return the function name and line number
	funcName := fn.Name()
	index := strings.LastIndex(funcName, ".")
	funcName = funcName[index+1:]
	return fileName, funcName, line
}
