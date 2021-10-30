package main

import (
	"fmt"
	"github.com/hoisie/mustache"
	"github.com/alecthomas/kingpin"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Configuration file
var configFile string

//Flags
var (
	updateFlag         = kingpin.Flag("update-list", "Update the list of templates and colorschemes").Bool()
	clearListFlag      = kingpin.Flag("clear-list", "Delete local master list caches").Bool()
	clearSchemesFlag   = kingpin.Flag("clear-templates", "Delete local scheme caches").Bool()
	clearTemplatesFlag = kingpin.Flag("clear-schemes", "Delete local template caches").Bool()
	configFileFlag     = kingpin.Flag("config", "Specify configuration file to use").Default("config.yaml").String()
)

//Configuration
var appConf SetterConfig

func main() {

	//Pase Flags
	kingpin.Version("1.0.0")
	kingpin.Parse()

	appConf = NewConfig(*configFileFlag)

	// appConf.Show()
	//TODO delete caches, if user wants to

	//Create cache paths, if missing
	p1 := filepath.Join(".", appConf.SchemesCachePath)
	os.MkdirAll(p1, os.ModePerm)
	p2 := filepath.Join(".", appConf.TemplatesCachePath)
	os.MkdirAll(p2, os.ModePerm)

	schemeList := LoadBase16ColorschemeList()
	templateList := LoadBase16TemplateList()

	if *updateFlag {
		schemeList.UpdateSchemes()
		templateList.UpdateTemplates()
	}

	scheme := schemeList.Find(appConf.Colorscheme)
	fmt.Println("[CONFIG]: Selected scheme: ", scheme.Name)

	for k, v := range appConf.Applications {
		if v.Enabled {
			schemeList = LoadBase16ColorschemeList()
			templateList = LoadBase16TemplateList()

			templ := templateList.Find(k)

			Base16Render(templ, scheme)
		}
	}

}

func Base16Render(templ Base16Template, scheme Base16Colorscheme) {

	fmt.Println("[RENDER]: Rendering template \"" + templ.Name + "\"")

	for k, v := range templ.Files {
		dir := appConf.TemplatesCachePath + templ.Name
		path := dir + "/" + k + ".mustache"
		os.MkdirAll(dir, os.ModePerm)

		// Create local template file, if not present
		if _, err := os.Stat(path); os.IsNotExist(err) {
			templFileData, err := DownloadFileToStirng(templ.RawBaseURL + "templates/" + k + ".mustache")
			check(err)
			saveFile, err := os.Create(path)
			//TODO delete old file?
			defer saveFile.Close()
			saveFile.Write([]byte(templFileData))
			saveFile.Close()
		}

		templFileDataBytes, err := ioutil.ReadFile(path)
		check(err)
		templFileData := string(templFileDataBytes)
		renderedFile := mustache.Render(templFileData, scheme.MustacheContext())

		// If this is a path with the right extension, we use the full path as output
		// If not, we treat it as a directory and use the same name as the template
		pathInput := appConf.Applications[templ.Name].Files[k]

		// Default to output folder instead of root '/' folder
		if pathInput == "" {
			pathInput = "./output/"
		}

		// Check whether we should treat this as a file  or directory path
		dirPath, fileName := filepath.Split(pathInput)
		if v.Extension != "" && filepath.Ext(fileName) != v.Extension {
			dirPath = pathInput
			fileName = k + v.Extension
		}

		os.MkdirAll(dirPath, os.ModePerm)
		savePath := filepath.Join(dirPath, fileName)

		//If DryRun is enabled, just print the output location for debugging
		if appConf.DryRun {
			fmt.Println("    - (dryrun) file would be written to: ", savePath)
		} else {
			switch appConf.Applications[templ.Name].Mode {
			case "rewrite":
				fmt.Println("     - writing: ", savePath)
				saveFile, err := os.Create(savePath)
				defer saveFile.Close()
				check(err)
				saveFile.Write([]byte(renderedFile))
				saveFile.Close()
			case "append":
				fmt.Println("     - appending to: ", savePath)
			case "replace":
				fmt.Println("     - replacing in: ", savePath)
			}
		}
	}

	if appConf.DryRun {
		fmt.Println("Not running hook, DryRun enabled: ", appConf.Applications[templ.Name].Hook)
	} else {
		exe_cmd(appConf.Applications[templ.Name].Hook)
	}
}

//TODO proper error handling
func check(e error) {
	if e != nil {
		panic(e)
	}

}
