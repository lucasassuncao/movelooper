package main

import (
	"io/fs"
	"movelooper/helper"
	"movelooper/logging"
	"movelooper/types"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alyu/configparser"
	"github.com/cheynewallace/tabby"
	"github.com/logrusorgru/aurora/v4"
)

var dir string

func getConfSections() (*configparser.Configuration, []string) {
	configparser.Delimiter = "="

	config, err := configparser.Read(types.ConfigFile)
	if err != nil {
		logging.Log.Errorf("Failed to read configuration file: %s. Error: %s.", types.ConfigFile, err.Error())
	}

	sections, err := config.AllSections()

	if err != nil {
		logging.Log.Errorf("Unable to read sections from the configuration file. Error: %s.", err.Error())
		return nil, nil
	}

	var availableSectionNames []string
	for _, section := range sections {
		availableSectionNames = append(availableSectionNames, section.Name())
	}
	return config, availableSectionNames
}

func moveFileAndAddLineToTabbyLog(files []fs.DirEntry, t *tabby.Tabby, entry, src string) {
	t.AddHeader("DATE", "TIME", "TYPE", "SOURCE", "DESTINATION", "NAME", "SIZE", "STATUS")
	for _, file := range files {
		fileInfo, _ := file.Info()

		if strings.HasSuffix(file.Name(), strings.ToUpper("."+entry)) || strings.HasSuffix(file.Name(), strings.ToLower("."+entry)) {
			status := helper.MoveFileToDestination(src+file.Name(), dir+"\\"+file.Name())
			t.AddLine(time.Now().Format("2006/01/02"), time.Now().Format("15:04:05"), entry, src, dir, file.Name(), helper.ByteCountDecimal(fileInfo.Size()), status)
		}
	}
	t.AddLine("")
}

func main() {
	logging.Log = logging.GetLogger("main")

	fd, _ := os.OpenFile(types.TabbyFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer fd.Close()

	w := tabwriter.NewWriter(fd, 0, 0, 4, ' ', 0)
	t := tabby.NewCustom(w)

	config, sectionNames := getConfSections()

	for _, sectionName := range sectionNames {

		if strings.Contains(sectionName, "global") {
			continue
		}

		section, err := config.Section(sectionName)

		if err != nil {
			logging.Log.Errorf("Failed to fetch section. Error: %s", err.Error())
		}

		src, dst := section.ValueOf("source"), section.ValueOf("destination")
		entries := strings.Split(section.ValueOf("entries"), ",")

		for _, entry := range entries {

			dir = dst + entry
			helper.CreateDirectory(dir)

			files, err := os.ReadDir(src)

			if err != nil {
				logging.Log.Errorf("Failed to read directory: %s. Error: %s.", src, err.Error())
			}

			var countFiles int = 0
			for _, file := range files {
				if file.Type().IsRegular() {
					if strings.HasSuffix(file.Name(), strings.ToUpper("."+entry)) || strings.HasSuffix(file.Name(), strings.ToLower("."+entry)) {
						countFiles++
					}
				}
			}

			switch countFiles {
			case 0:
				if types.LogType == "logs" {
					logging.Log.Infof("No .%s file(s) to move.", entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("No .%s file(s) to move.", aurora.Yellow(entry)))
				}
			case 1:
				if types.LogType == "logs" {
					logging.Log.Infof("%d file .%s to move", countFiles, entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("%d file .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				}
				moveFileAndAddLineToTabbyLog(files, t, entry, src)
			default:
				if types.LogType == "logs" {
					logging.Log.Infof("%d files .%s to move", countFiles, entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("%d file .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				}
				moveFileAndAddLineToTabbyLog(files, t, entry, src)
			}
		}

	}
}
