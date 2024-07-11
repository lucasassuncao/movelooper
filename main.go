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
		logging.Log.Errorf("Falha ao ler arquivo de configuracao: %s. Erro: %s.", types.ConfigFile, err.Error())
	}

	sections, err := config.AllSections()

	if err != nil {
		logging.Log.Errorf("Não foi possível ler as Secoes do arquivo de configuracao. Erro: %s.", err.Error())
		return nil, nil
	}

	var availableSectionNames []string
	for _, section := range sections {
		availableSectionNames = append(availableSectionNames, section.Name())
	}
	return config, availableSectionNames
}

func moveFileAndAddLineToTabbyLog(files []fs.DirEntry, t *tabby.Tabby, entry, src string) {
	t.AddHeader("DATA", "HORA", "TIPO", "ORIGEM", "DESTINO", "NOME", "TAMANHO", "STATUS")
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

		// logging.Log.Infof(aurora.Sprintf("Processando seção: %s", aurora.Yellow(sectionName)))

		section, err := config.Section(sectionName)

		if err != nil {
			logging.Log.Errorf("Falha ao obter seção. Erro: %s", err.Error())
		}

		src, dst := section.ValueOf("source"), section.ValueOf("destination")
		entries := strings.Split(section.ValueOf("entries"), ",")

		for _, entry := range entries {

			dir = dst + entry
			helper.CreateDirectory(dir)

			files, err := os.ReadDir(src)

			if err != nil {
				logging.Log.Errorf("Falha ao ler o diretorio: %s. Erro: %s.", src, err.Error())
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
				logging.Log.Infof(aurora.Sprintf("Nenhum arquivo .%s para mover", aurora.Yellow(entry)))
			case 1:
				logging.Log.Infof(aurora.Sprintf("%d arquivo .%s para mover", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				moveFileAndAddLineToTabbyLog(files, t, entry, src)
			default:
				logging.Log.Infof(aurora.Sprintf("%d arquivos .%s para mover", aurora.Yellow(countFiles), aurora.Yellow(entry)))
				moveFileAndAddLineToTabbyLog(files, t, entry, src)
			}
		}

	}
	// fmt.Print("Aperte 'Enter' para continuar...")
	// bufio.NewReader(os.Stdin).ReadBytes('\n')
}
