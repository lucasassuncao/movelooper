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

type ConfigurationManager struct {
	Config       *configparser.Configuration
	SectionNames []string
	TabbyWriter  *tabby.Tabby
	TabbyFile    string
	Dir          string
	LogType      string
}

func NewConfigManager(tabbyFile, logType string) (*ConfigurationManager, error) {
	config, sectionNames := getConfSections()

	fd, err := os.OpenFile(tabbyFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	w := tabwriter.NewWriter(fd, 0, 0, 4, ' ', 0)
	t := tabby.NewCustom(w)

	return &ConfigurationManager{
		Config:       config,
		SectionNames: sectionNames,
		TabbyWriter:  t,
		TabbyFile:    tabbyFile,
		LogType:      logType,
	}, nil
}

func getConfSections() (*configparser.Configuration, []string) {
	configparser.Delimiter = "="

	config, err := configparser.Read(types.ConfigFile)
	if err != nil {
		logging.Log.Errorf("Failed to read configuration file: %s. Error: %s.", types.ConfigFile, err.Error())
		return nil, nil
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

func (cm *ConfigurationManager) moveFileAndAddLineToTabbyLog(files []fs.DirEntry, entry, src string) {
	cm.TabbyWriter.AddHeader("DATE", "TIME", "TYPE", "SOURCE", "DESTINATION", "NAME", "SIZE", "STATUS")
	for _, file := range files {
		fileInfo, _ := file.Info()

		if strings.HasSuffix(file.Name(), strings.ToUpper("."+entry)) || strings.HasSuffix(file.Name(), strings.ToLower("."+entry)) {
			status := helper.MoveFileToDestination(
				src+file.Name(),
				cm.Dir+"\\"+file.Name(),
			)

			cm.TabbyWriter.AddLine(
				time.Now().Format("2006/01/02"),
				time.Now().Format("15:04:05"),
				entry,
				src,
				cm.Dir,
				file.Name(),
				helper.ByteCountDecimal(fileInfo.Size()),
				status,
			)
		}
	}
	cm.TabbyWriter.AddLine("")
}

func (cm *ConfigurationManager) processSections() {
	for _, sectionName := range cm.SectionNames {
		if strings.Contains(sectionName, "global") {
			continue
		}

		section, err := cm.Config.Section(sectionName)
		if err != nil {
			logging.Log.Errorf("Failed to fetch section. Error: %s", err.Error())
			continue
		}

		src, dst := section.ValueOf("source"), section.ValueOf("destination")
		entries := strings.Split(section.ValueOf("entries"), ",")

		for _, entry := range entries {
			cm.Dir = dst + entry
			helper.CreateDirectory(cm.Dir)

			files, err := os.ReadDir(src)
			if err != nil {
				logging.Log.Errorf("Failed to read directory: %s. Error: %s.", src, err.Error())
				continue
			}

			var countFiles int
			for _, file := range files {
				if file.Type().IsRegular() {
					if strings.HasSuffix(file.Name(), strings.ToUpper("."+entry)) || strings.HasSuffix(file.Name(), strings.ToLower("."+entry)) {
						countFiles++
					}
				}
			}

			switch countFiles {
			case 0:
				if cm.LogType == "logs" {
					logging.Log.Infof("No .%s file(s) to move.", entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("No .%s file(s) to move.", aurora.Yellow(entry)))
				}
			case 1:
				if cm.LogType == "logs" {
					logging.Log.Infof("%d file .%s to move", countFiles, entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("%d file .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				}
				cm.moveFileAndAddLineToTabbyLog(files, entry, src)
			default:
				if cm.LogType == "logs" {
					logging.Log.Infof("%d files .%s to move", countFiles, entry)
				} else {
					logging.Log.Infof(aurora.Sprintf("%d file .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				}
				cm.moveFileAndAddLineToTabbyLog(files, entry, src)
			}
		}
	}
}

func main() {
	logging.Log = logging.GetLogger("main")

	configManager, err := NewConfigManager(types.TabbyFile, types.LogType)
	if err != nil {
		logging.Log.Errorf("Failed to initialize ConfigManager. Error: %s", err.Error())
		return
	}

	configManager.processSections()
}
