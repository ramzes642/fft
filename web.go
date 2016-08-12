package main
import (
    "net/http"
    "flag"
    "log"
    "fmt"
    "io"
    "os"
    "html/template"
    "io/ioutil"
)

const PATH_WWW = "www/";

var web bool;
var webport string;

func main() {
    flag.IntVar(&windowSize, "samples", 512, "Window size")
    flag.IntVar(&graphSize, "graph", 1, "Graph width size")
    flag.IntVar(&sampleRate, "rate", 16000, "Sample rate")
    flag.BoolVar(&exportRaw, "raw", false, "Do not convert to amp/phase")
    flag.BoolVar(&exportPng, "png", true, "Export png")
    flag.BoolVar(&exportTone, "tone", false, "Export tone info")
    flag.BoolVar(&windowHann, "hann", false, "Apply hann window")
    flag.BoolVar(&web, "web", false, "Start web server")
    flag.StringVar(&webport, "webport", ":8181", "Web port binding")

    flag.Parse();

    if !web {
        if flag.NArg() == 0 && !web {
            flag.PrintDefaults();
            return;
        }

        filename := flag.Arg(0);
        if filename[len(filename)-3:] == "wav" {
            decode(filename);
        }
        if filename[len(filename)-3:] == "tsv" {
            encode(filename);
        }

    } else {

        log.Printf("Web server started on %s\n", webport);

        // HTTP
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
            http.ServeFile(w, r, PATH_WWW + r.URL.Path)
        });

        http.HandleFunc("/upload", upload);
        http.HandleFunc("/spec", make_spec);


        http.ListenAndServe(webport, nil)
    }
}

// upload logic
func upload(w http.ResponseWriter, r *http.Request) {
    fmt.Println("method:", r.Method)
    if r.Method == "GET" {
        w.Header().Set("Content-type", "text/html; charset=utf8");

        files, _ := ioutil.ReadDir(PATH_WWW + "upload/")

        t, _ := template.ParseFiles("tpl/upload.tpl")
        t.Execute(w, files)
    } else {
        r.ParseMultipartForm(32 << 20)
        file, handler, err := r.FormFile("f")
        if err != nil {
            fmt.Println(err)
            return
        }
        defer file.Close()
        //w.Header().Set("Location", "/upload");
        http.Redirect(w, r, "/upload", 302)

        //fmt.Fprintf(w, "%v", handler.Header)
        f, err := os.OpenFile(PATH_WWW + "upload/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
        if err != nil {
            fmt.Println(err)
            return
        }
        defer f.Close()
        io.Copy(f, file)
    }
}

func make_spec(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        w.Header().Set("Content-type", "image/png");
        filename := r.URL.Query().Get("f")

        if filename[len(filename)-3:] == "wav" {
            exportPng = true;
            decode(PATH_WWW + filename);
        }
        http.Redirect(w, r, filename + ".png", 302)
        //io.Copy(w, filename + ".png");
    }
}