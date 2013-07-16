package kwiscale

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
    "reflect"
)

type RouteMap struct {
	Route   *regexp.Regexp
	Handler IRequestHandler
}

// handlers stack
var globalhandlers []RouteMap

// sessions handler
var sessions map[string]map[string]interface{}

// add an hancler to the stack
func AddHandler(r IRequestHandler) {
    field,_ := reflect.TypeOf(r).Elem().FieldByName("RequestHandler")
    route := field.Tag.Get("route")
    log.Printf("Append route: %s", route)
    reg := regexp.MustCompile(route)
    routemap := RouteMap{reg, r}
    globalhandlers = append(globalhandlers, routemap)
}

// start to serve on given address:port
func Serve(address string) {
	http.Handle("/statics/", http.StripPrefix("/statics", http.FileServer(http.Dir("./statics"))))
	http.HandleFunc("/", dispatch)
	http.ListenAndServe(address, nil)
}

// dispatch request to correct handler
func dispatch(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("ERROR", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
		}
	}()

	r.ParseForm()
	rcall := r.URL.Path

	for _, route := range globalhandlers {
		if res := route.Route.FindStringSubmatch(rcall); len(res) > 0 {

			handler := route.Handler
			//handler := _handler

			if len(res) > 1 {
				// params captured
				handler.setParams(w, r, res[1:])
			} else {
				handler.setParams(w, r, nil)
			}

			switch r.Method {
			case "GET":
				handler.Get()
			case "POST":
				handler.Post()
			case "DELETE":
				handler.Delete()
			case "HEAD":
				handler.Head()
			case "PUT":
				handler.Put()
			default:
				panic("Method not found: " + r.Method)
			}
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)

}
