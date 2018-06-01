package runtime

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// FunctionKey holds the function field
const FunctionKey = "function"

// PackageKey holds the package field
const PackageKey = "package"

// LineKey holds the line field
const LineKey = "line"

// FileKey holds the file field
const FileKey = "file"

// Formatter decorates log entries with function name and package name (optional) and line number (optional)
type Formatter struct {
	ChildFormatter logrus.Formatter
	// When true, line number will be tagged to fields as well
	Line bool
	// When true, package name will be tagged to fields as well
	Package bool
	// When true, file name will be tagged to fields as well
	File bool
}

// Format the current log entry by adding the function name and line number of the caller.
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	function, file, line := f.getCurrentPosition(entry)

	packageEnd := strings.LastIndex(function, ".")
	functionName := function[packageEnd+1:]

	data := logrus.Fields{FunctionKey: functionName}
	if f.Line {
		data[LineKey] = line
	}
	if f.Package {
		packageName := function[:packageEnd]
		// parenPosition := strings.LastIndex(packageName, "(")
		// if parenPosition != -1 {
		// 	packageName = packageName[:parenPosition-1]
		// }
		data[PackageKey] = packageName
	}
	if f.File {
		data[FileKey] = file
	}
	for k, v := range entry.Data {
		data[k] = v
	}
	entry.Data = data

	return f.ChildFormatter.Format(entry)
}

func (f *Formatter) getCurrentPosition(entry *logrus.Entry) (string, string, string) {
	skip := 6
	if len(entry.Data) == 0 {
		skip = 8
	}
start:
	pc, file, line, _ := runtime.Caller(skip)
	lineNumber := ""
	if f.Line {
		lineNumber = fmt.Sprintf("%d", line)
	}
	function := runtime.FuncForPC(pc).Name()
	if function == "reflect.callMethod" {
		skip -= 2
		goto start
	}
	if strings.HasPrefix(function, "runtime.call") {
		skip--
		goto start
	}
	return function, file, lineNumber
}
