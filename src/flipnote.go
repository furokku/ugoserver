package main

import (
    "time"
    "os"
    "fmt"
    "log"
)

type (
    flipnote struct {
        id int
        author string
        filename string
        uploaded_at time.Time
    }

    tmb []byte
)

var (
    tmbSize int = 0x6A0 // size of tmb data
    lockOffset int = 0x10 // flipnote lock state
)

// Get the TMB for a given flipnote
// Used for flipnote previews on menu type 2
func getTmbData(fn string) tmb {
    buf := make([]byte, tmbSize)
    path := fmt.Sprintf(dataPath + "flipnotes/%s.ppm", fn)

    file, err := os.Open(path)
    if err != nil {
        log.Fatalf("getTmbData: %v", err)
    }

    defer file.Close()

    n, err := file.Read(buf)
    if err != nil {
        log.Fatalf("getTmbData: %v", err)
    } else if n != tmbSize {
        log.Printf("getTmbData: WARNING: read %v bytes instead of 1696", n)
    }

    return buf
}


// return whether a flipnote is locked
// 0 if not, 1 if it is
func (t tmb) flipnoteIsLocked() int {
    l := int(t[ lockOffset ])

    if l != 0 && l != 1 {
        log.Printf("flipnoteIsLocked: WARNING: invalid lock state; returning 0")
        return 0
    } else {
        return l
    }
}
