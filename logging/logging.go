package logging
// poor man's leveled logger

import (
    "log"
    "os"
    "io/ioutil"
)

var (
    Trace   *log.Logger
    Info    *log.Logger
    Warning *log.Logger
    Error   *log.Logger
)

func init() {
    //traceHandle := os.Stdout
    traceHandle := ioutil.Discard
    infoHandle := ioutil.Discard
    warningHandle := os.Stdout
    errorHandle := os.Stderr

    Trace = log.New(traceHandle,
        "TRACE: ",
        log.Ldate|log.Ltime)//|log.Lshortfile)

    Info = log.New(infoHandle,
        " INFO: ",
        log.Ldate|log.Ltime)

    Warning = log.New(warningHandle,
        " WARN: ",
        log.Ldate|log.Ltime)

    Error = log.New(errorHandle,
        "ERROR: ",
        log.Ldate|log.Ltime)
}
