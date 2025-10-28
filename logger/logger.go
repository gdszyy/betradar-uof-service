package logger

import (
	"log"
	"os"
)

var (
	// Info 正常日志，输出到 stdout (显示为 [info])
	Info *log.Logger
	
	// Error 错误日志，输出到 stderr (显示为 [err])
	Error *log.Logger
)

func init() {
	// 初始化 Info logger (输出到 stdout)
	Info = log.New(os.Stdout, "", log.LstdFlags)
	
	// 初始化 Error logger (输出到 stderr)
	Error = log.New(os.Stderr, "", log.LstdFlags)
}

// Println 输出正常日志到 stdout
func Println(v ...interface{}) {
	Info.Println(v...)
}

// Printf 格式化输出正常日志到 stdout
func Printf(format string, v ...interface{}) {
	Info.Printf(format, v...)
}

// Errorln 输出错误日志到 stderr
func Errorln(v ...interface{}) {
	Error.Println(v...)
}

// Errorf 格式化输出错误日志到 stderr
func Errorf(format string, v ...interface{}) {
	Error.Printf(format, v...)
}

// Fatalf 输出致命错误并退出程序
func Fatalf(format string, v ...interface{}) {
	Error.Fatalf(format, v...)
}

