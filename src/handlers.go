package main


import (
    "os"
    "io"

    "floc/ugoserver/ugo"
    "fmt"

    "github.com/gorilla/mux"
    "net/http"

    "encoding/base64"
    "encoding/binary"
    "encoding/hex"
    "strconv"
    "strings"
)


// probably needs a rewrite
// No UGOs are requested at the root so this can go
// Doesn't send any special headers either so not really
// important for logging
// func handleUgo(w http.ResponseWriter, r *http.Request) {
// 
//     log.Printf("%v requested %v%v with header %v\n", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)
// 
//     // there was method checking code here
//     // but gorilla mux exists
// 
//     vars := mux.Vars(r)
//     switch vars["ugo"] {
// 
//         case "index":
//             w.Write(indexUGO.Pack())
//         default:
//             w.WriteHeader(http.StatusNotFound)
//     }
// 
//     return
// }


// Replace this (eventually)
/* func returnFromFs(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL, r.Header)

    // Why did I even do this this way?
    // This is stupid and should be replaced with
    // a different handler entirely
    // 
    // This approach is stupid and insecure and blehhh!!!
    // but for now it works so I'll put it off

    // TODO: The eula and etc files should probably be read and stored
    // ~~within the server~~ elsewhere
    // That would allow the base files to be in utf8 and get rid of
    // essentially the empty folders that are in
    // hatena/static/ds/ and get rid of some code i dont like
    //
    // done
    data, err := os.ReadFile(dataPath + r.URL.Path)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        log.Printf("response 404 at %v%v (file handler): %v", r.Host, r.URL.Path, err)
        return
    }

    w.Write(data)
    log.Printf("response 200 at %v%v with headers %v", r.Host, r.URL.Path, w.Header())
} */

// Not my finest code up there so we're doing this a better way
func serveFlipnotes(w http.ResponseWriter, r *http.Request) {

    infolog.Printf("%v requested %v%v with header %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)

    id := vars["id"]
    ext := vars["ext"]

    path := "/flipnotes/" + id + ".ppm"

    switch ext {
        case "ppm":
            data, err := os.ReadFile(configuration.HatenaDir + "/hatena_storage" + path)
            if err != nil {
                w.WriteHeader(http.StatusNotFound)
                infolog.Printf("%v got 404 at %v%v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path)
                return
            }

            w.Write(data)
//          log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
            return

        case "htm":
            if fi, err := os.Stat(configuration.HatenaDir + "/hatena_storage" + path); err == nil {
                w.Write([]byte(fmt.Sprintf("<html><head><meta name=\"upperlink\" content=\"%s\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"%s\"></head><body><p>wip<br>obviously this would be unfinished<br><br>debug:<br>file: %s<br>size: %d<br>modified: %s</p></body></html>", configuration.ServerUrl+path, configuration.ServerUrl+path, id, fi.Size(), fi.ModTime())))
                return
            } else {
                w.WriteHeader(http.StatusNotFound)
                infolog.Printf("%s got 404 at %s%s : %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
                return
            }

        case "info":
            w.Write([]byte{0x30, 0x0A, 0x30, 0x0A}) // write 0\n0\n because flipnote is weird
            return

        default:
            w.WriteHeader(http.StatusNotFound)
            return
    }
}


// Handler for building ugomenus for the front page
// recent, hot, most liked, etc..
func serveFrontPage(w http.ResponseWriter, r *http.Request) {

    infolog.Printf("%s requested %s%s with header %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)
    
    vars := mux.Vars(r)
    base := gridBaseUGO

    pageType := vars["type"]
    pageQ := r.URL.Query().Get("page")

    page, err := strconv.Atoi(pageQ)
    if pageQ == "" {
        // do NOT print error message if the query is empty
        page = 1
    } else if err != nil {
        // When the page isn't specified this should be expected
        // TODO: get rid of this under above condition: done
        infolog.Printf("%s passed invalid page to %s%s: %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
        page = 1
    }

    flipnotes, total := getFrontFlipnotes(pageType, page)

    // Add top screen titles
    base.Entries = append(base.Entries, ugo.MenuEntry{
        EntryType: 1,
        Data: []string{
            "0",
            base64.StdEncoding.EncodeToString(encUTF16LE("Front page")),
            base64.StdEncoding.EncodeToString(encUTF16LE(fmt.Sprintf("Page %d / %d", page, countPages(total)))),
        },
    })
    base.Entries = append(base.Entries, ugo.MenuEntry{
        EntryType: 2, // category
        Data: []string{
            configuration.ServerUrl + "/front/recent.uls",
            base64.StdEncoding.EncodeToString(encUTF16LE(pageType)),
            "1",
        },
    })

    if page > 1 {
        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 4,
            Data: []string{
                fmt.Sprintf(configuration.ServerUrl + "/front/%s.uls?page=%d", pageType, page-1),
                "100",
                base64.StdEncoding.EncodeToString(encUTF16LE("Previous page")),
            },
        })
    }

    for _, f := range flipnotes {
        tempTmb := f.getTmb()
        if tempTmb == nil {
            warnlog.Printf("tmb is nil")
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 4,
            Data: []string{
                fmt.Sprintf(configuration.ServerUrl + "/flipnotes/%s.ppm", f.filename),
                "3",
                "0",
                "420", // star counter (TODO)
                fmt.Sprint(tempTmb.flipnoteIsLocked()),
                "0", // ??
            },
        })

        base.Embed = append(base.Embed, tempTmb)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    // TODO: add previous/next page buttons
    data := base.Pack()
    //fmt.Println(string(data))
    w.Write(data)
}



// I have no idea why this is needed
// nor what it does
// Changes some statistic in the flipnote viewer maybe?
// Replaced by a catchall function
/* func handleInfo(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("0\n0\n"))
} */


// Return delete, upload, download, eula
func handleEula(w http.ResponseWriter, r *http.Request) {
    infolog.Printf("%v requested %v%v with header %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)
    txt := vars["txt"]

    // if !slices.Contains(txts, file) {
    //    http.Error(w, "not found", http.StatusNotFound)
    //    return
    //}
    
    text, err := os.ReadFile(configuration.HatenaDir + "/static/txt/" + txt + ".txt")
    if err != nil {
        warnlog.Printf("failed to read %v: %v", txt, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}


// Simply log the request and do nothing
// Replaced by a catchall function
/* func sendWip(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)
    ppmPath := serverUrl + "/flipnotes/" + vars["filename"] + ".ppm"

    w.Write([]byte("<html><head><meta name=\"upperlink\" content=\"" + ppmPath + "\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"" + ppmPath + "\"></head><body><p>wip<br>obviously this would be unfinished</p></body></html>"))
} */

// accept flipnotes uploaded thru internal ugomemo:// url
// or flipnote.post url
func postFlipnote(w http.ResponseWriter, r *http.Request) {

    infolog.Printf("%v requested %v%v with header %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)

    // make sure request has a valid SID
    // we don't want a flood of random flipnotes
    // after all...
    session, ok := sessions[r.Header.Get("X-Dsi-Sid")]
    if !ok {
        infolog.Printf("unauthorized attempt to post flipnote")
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    ppmBody, err := io.ReadAll(r.Body)
    if err != nil {
        warnlog.Printf("failed to read ppm from POST request body! %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    // should this stay? it may be simpler to just store flipnotes
    // by their id in the database
    filename := strings.ToUpper(hex.EncodeToString(ppmBody[0x78 : 0x7B])) + "_" +
                string(ppmBody[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppmBody[0x88 : 0x90]))

    debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, filename)

    fp, err := os.OpenFile(configuration.HatenaDir + "/hatena_storage/flipnotes/" + filename + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        // Realistically, two flipnote filenames shouldn't clash.
        // if it becomes an issue, I will either save them in reference
        // to their id in the database or start adding randomized
        // characters in the end
        //
        // 26/01/24 - if somebody tries to upload the same flipnote twice,
        // this becomes a problem -- idk how I didn't think about this earlier
        // it may be better to store them with their id as the name
        // as this would eliminate filename clashes and the original
        // filename is stored within the ppm body itself
        warnlog.Printf("failed to open path to ppm: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer func() {
        if err := fp.Close(); err != nil {
            panic(err)
        }
    }()

    if _, err := fp.Write(ppmBody); err != nil {
        warnlog.Printf("failed to write ppm to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    if _, err := db.Exec("INSERT INTO flipnotes (author_id, filename) VALUES ($1, $2)", session.fsid, filename); err != nil {
        warnlog.Printf("failed to update database: %v", err)
    }

    w.WriteHeader(http.StatusOK)
}
