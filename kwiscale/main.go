/*
Kwiscale command line interface.
*/
package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
)

var GOPATH = os.Getenv("GOPATH")

const (
	PROJECT_OPT     = "project"      // argument to set project name
	HANDLER_OPT     = "handlers"     // argument to set handlers name
	PROJECT_DEFAULT = "kwiscale-app" // default project name
	HANDLER_DEFAULT = "handlers"     // default handlers package name
)

type ymlstruct map[string]interface{}

func main() {

	out := cli.StringFlag{
		Name:   PROJECT_OPT,
		Value:  PROJECT_DEFAULT,
		Usage:  "project name, will set " + GOPATH + "/src/[projectname]",
		EnvVar: "KWISCALE_PROJECT",
	}

	handlers := cli.StringFlag{
		Name:   HANDLER_OPT,
		Value:  HANDLER_DEFAULT,
		Usage:  "handlers package name",
		EnvVar: "KWISCALE_HANDLERS",
	}

	app := cli.NewApp()
	app.Flags = []cli.Flag{out, handlers}
	app.Name = "kwiscale"
	app.Usage = "tool to manage kwiscale application"
	app.Version = "0.0.1"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:  "new",
			Usage: "Generate resources (application, handlers...)",
			Subcommands: []cli.Command{
				{
					Name:   "app",
					Usage:  "Create application",
					Action: newApplication,
				},
				{
					Name:   "handler",
					Usage:  "Create handler",
					Action: newHandler,
				},
			},
		},
		{
			Name:   "generate",
			Usage:  "Parse configuration and generate handlers, main file...",
			Action: parseConfig,
		},
	}

	app.Run(os.Args)
}

// create an application
func newApplication(c *cli.Context) {
	out := getProjectPath(c)
	log.Println("Create application in directory:", out)

	createDirectories(out, c.GlobalString(HANDLER_OPT))
	createConfig(c)
	createApp(c)
}

// add handler in config.yml
func newHandler(c *cli.Context) {
	var (
		y           = loadYaml(c)
		hpkg        = c.GlobalString(HANDLER_OPT)
		name        = c.Args().First() //handler short name
		realname    = c.Args().Get(2)  //alias if any
		handlername = hpkg + "." + strings.Title(name) + "Handler"
		route       = c.Args().Get(1)
		m           = map[string]string{} //a simple map to handle route for template
	)

	// create handler file
	createHandlerFile(handlername, c.GlobalString(HANDLER_OPT), getProjectPath(c))

	// change configuration file
	m["handler"] = handlername
	if realname != "" {
		m["alias"] = realname
	}

	if _, ok := y["routes"]; !ok {
		y["routes"] = make(map[interface{}]interface{})
	}

	// append handler
	routes := y["routes"].(map[interface{}]interface{})

	routes[route] = m

	b, _ := yaml.Marshal(y)
	cfg := filepath.Join(getProjectPath(c), "config.yml")
	ioutil.WriteFile(cfg, b, 0666)
}

// parse config file to generate handler file in handlers package
// by calling createHandlerFile, then
// call addHandlerInApp to append handlers in main.go
func parseConfig(c *cli.Context) {
	y := loadYaml(c)

	if _, ok := y["routes"]; !ok {
		log.Fatal("There are no route to create in configuration file")
	}

	routes := y["routes"]
	handlers := make([]map[string]string, 0)
	for route, v := range routes.(map[interface{}]interface{}) {
		handlername := v.(map[interface{}]interface{})["handler"].(string)
		alias := ""
		if a, ok := v.(map[interface{}]interface{})["alias"]; ok {
			alias = a.(string)
		}
		createHandlerFile(handlername, c.GlobalString(HANDLER_OPT), getProjectPath(c))
		handlers = append(handlers, map[string]string{
			"handler": handlername,
			"route":   route.(string),
			"alias":   alias,
		})
	}
}

// create handler file.
func createHandlerFile(handler, handlerpkg, where string) {
	var (
		parts    = strings.Split(handler, ".")
		name     = parts[1]
		filename = strings.ToLower(strings.Replace(name, "Handler", "", -1))
		path     = filepath.Join(where, handlerpkg, filename+".go")
	)

	if _, err := os.Stat(path); err == nil {
		log.Println("Handler file already exists:", path)
		return
	}

	tpl, _ := template.New("handler").Parse(TPLHANDLER)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	tpl.Execute(f, map[string]string{
		"Handler":     name,
		"HandlersPKG": handlerpkg,
	})

	log.Println("Handler created:", path)
}

// create directory tree
func createDirectories(out, hpkg string) {
	for _, p := range []string{
		hpkg,
		"templates",
		"statics",
	} {

		path := filepath.Join(out, p)
		if err := os.MkdirAll(path, 0776); err != nil {
			log.Println(err)
		}
	}
}

// create a config.yml file
func createConfig(c *cli.Context) {
	out := getProjectPath(c)
	appname := c.GlobalString(PROJECT_OPT)
	p := filepath.Join(out, "config.yml")

	y := ymlstruct{
		"listen":    ":8000",
		"staticdir": "./statics",
		"session": map[string]string{
			"secret": "Change this to a secret passphrase",
			"name":   appname,
		},
		"template": map[string]string{
			"dir": "./templates",
		},
	}
	b, _ := yaml.Marshal(y)

	ioutil.WriteFile(p, b, 0666)
}

// create application in project
func createApp(c *cli.Context) {
	out := getProjectPath(c)
	p := filepath.Join(out, "main.go")

	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()

	tpl, _ := template.New("main").Parse(TPLAPP)
	tpl.Execute(f, map[string]string{
		"Project":     c.GlobalString(PROJECT_OPT),
		"HandlersPKG": c.GlobalString(HANDLER_OPT),
	})

}

// returns project path
func getProjectPath(c *cli.Context) string {
	to := c.GlobalString(PROJECT_OPT)
	return filepath.Join(GOPATH, "src", to)
}

// load config.yml file
func loadYaml(c *cli.Context) ymlstruct {
	out := getProjectPath(c)
	out = filepath.Join(out, "config.yml")
	b, err := ioutil.ReadFile(out)
	if err != nil {
		log.Fatal(err)
	}

	y := ymlstruct{}
	yaml.Unmarshal(b, y)
	return y
}
