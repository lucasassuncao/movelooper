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
	"github.com/pterm/pterm"
)

type ConfigurationManager struct {
	Config       *configparser.Configuration
	SectionNames []string
	TabbyWriter  *tabby.Tabby
	TabbyFile    string
	Dir          string
}

type SectionFields struct {
	Entries     []string
	Source      string
	Destination string
}

func NewConfigManager(tabbyFile string) (*ConfigurationManager, error) {
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
	}, nil
}

func getConfSections() (*configparser.Configuration, []string) {
	configparser.Delimiter = "="

	config, err := configparser.Read(types.ConfigFile)
	if err != nil {
		logging.Logger.Error(aurora.Sprintf("Failed to read configuration file: %s. Error: %s", types.ConfigFile, err.Error()))
		return nil, nil
	}

	sections, err := config.AllSections()
	if err != nil {
		logging.Logger.Error(aurora.Sprintf("Unable to read sections from the configuration file. Error: %s", err.Error()))
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

	sf := &SectionFields{} // Set up a pointer to a SectionFields initialized with ZERO Values

	for _, sectionName := range cm.SectionNames {
		if strings.Contains(sectionName, "global") {
			continue
		}

		section, err := cm.Config.Section(sectionName)
		if err != nil {
			logging.Logger.Error(aurora.Sprintf("Failed to fetch section. Error: %s", err.Error()))
			continue
		}

		sf.Source = section.ValueOf("source")
		sf.Destination = section.ValueOf("destination")
		sf.Entries = strings.Split(section.ValueOf("entries"), ",")

		for _, entry := range sf.Entries {
			cm.Dir = sf.Destination + entry
			helper.CreateDirectory(cm.Dir)

			files, err := os.ReadDir(sf.Source)
			if err != nil {
				logging.Logger.Error(aurora.Sprintf("Failed to read directory: %s. Error: %s", sf.Source, err.Error()))
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
				logging.Logger.Info(aurora.Sprintf("No .%s file(s) to move", aurora.Yellow(entry)))
			case 1:
				logging.Logger.Info(aurora.Sprintf("%d file .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				cm.moveFileAndAddLineToTabbyLog(files, entry, sf.Source)
			default:
				logging.Logger.Info(aurora.Sprintf("%d files .%s to move", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				cm.moveFileAndAddLineToTabbyLog(files, entry, sf.Source)
			}
		}
	}
}

func main() {
	// Initialize the logger with custom configuration
	config := logging.LoggerConfig{
		LogType:       "terminal",
		LogLevel:      pterm.LogLevelInfo,
		IncludeCaller: false,
	}

	logging.Logger = logging.ConfigureLogger(config)

	configManager, err := NewConfigManager(types.TabbyFile)
	if err != nil {
		logging.Logger.Error(aurora.Sprintf("failed to initialize configManager. Error: %s", err.Error()))
		return
	}

	configManager.processSections()
}
