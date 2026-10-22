package main

import (
    "io"
    "log"
    "net/http"
    "slices"
    "strings"
)

const colorReset string = "\033[0m"

var warnlog, errorlog, infolog, debuglog, reqlog, resplog *log.Logger

func wrapResponseWriter(w http.ResponseWriter) *rwWrapper {
    return &rwWrapper{ResponseWriter: w}
}

func (rw *rwWrapper) Status() int {
    return rw.status
}

func (rw *rwWrapper) WriteHeader(code int) {
    if rw.done {
        return
    }

    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
    rw.done = true
}

// logger is middleware to log all HTTP requests and responses
func (e *env) loggerMiddleware(next http.Handler) http.Handler {
    fn := func(w http.ResponseWriter, r *http.Request) {
        ua := r.Header.Get("User-Agent")

        // ignore bots and etc.
        if e.cnf.UseHosts { // && strings.HasPrefix(r.URL.Path, "/ds") { -- There really isn't any other good way to stop log spam other than only allowing proper domains
            if !slices.Contains(e.cnf.Hosts, r.Host) || (strings.HasPrefix(r.URL.Path, "/ds") && ua != "" && ua != "Nitro WiFi SDK/5.1") {
                w.Write([]byte("you may not access this resource in this way"))
                return
            }
        }
        //log request
        // the bots win. I am only going to log valid requests to /ds/ endpoints
        if strings.HasPrefix(r.URL.Path, "/ds") {
            reqlog.Printf("%v %v %v%v // %v", r.Header.Get("X-Real-Ip"), r.Method, r.Host, r.RequestURI, r.Header)
        }

        rww := wrapResponseWriter(w)
        next.ServeHTTP(rww, r)
        //log response
        // same as before
        if strings.HasPrefix(r.URL.Path, "/ds") {
            resplog.Printf("%v %v %v%v %v // %v", rww.status, r.Method, r.Host, r.RequestURI, r.Header.Get("X-Real-Ip"), w.Header())
        }
    }

    return http.HandlerFunc(fn)
}


// make all the loggers and stuff
func initlog(out io.Writer) {
    debuglog = log.New(out, "[debug] ", log.Lshortfile|log.Ldate|log.Ltime)
    infolog  = log.New(out, "[info] ", log.Ldate|log.Ltime)
    reqlog   = log.New(out, "[req] ", log.Ldate|log.Ltime)
    resplog  = log.New(out, "[resp] ", log.Ldate|log.Ltime)
    warnlog  = log.New(out, "\033[33m[warn] "+colorReset, log.Ldate|log.Ltime)
    errorlog = log.New(out, "\033[31m[error] "+colorReset, log.Lshortfile|log.Ldate|log.Ltime)
}