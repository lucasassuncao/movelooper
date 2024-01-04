package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Delta456/box-cli-maker/v2"
	"github.com/alyu/configparser"
	"github.com/cheynewallace/tabby"
)

/*===================================================================
	CONSTANTS AND VARS
===================================================================*/

var dir string

const logFile = "D:\\Scripts\\go\\log\\movelooper.log"
const tabbyFile = "D:\\Scripts\\go\\log\\movelooper_table_output.log"

const configFile = "D:\\Scripts\\go\\conf\\movelooper.ini"

/*===================================================================
	LOG FUNCTION
===================================================================*/

func logger(errorlevel, message string) {
	logfile, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Não foi possível abrir o arquivo de log " + err.Error())
	}
	defer logfile.Close()
	logfile.WriteString(time.Now().Format("2006/01/02 15:04:05") + " [" + getPID() + "] " + errorlevel + " " + message + "\n")
}

/*===================================================================
	FUNCTIONS
===================================================================*/
// Get the Script's PID for use in the LOGGER function+
func getPID() string {
	pid := os.Getpid()
	pidConverted := strconv.FormatInt(int64(pid), 10)

	return pidConverted
}

func createDirectory(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			logger("[ERROR]", fmt.Sprintf("Falha ao criar diretorio: %s. Erro: %s.", dir, err.Error()))
		} else {
			logger("[INFO]", fmt.Sprintf("Sucesso ao criar diretorio: %s.", dir))
		}
	}
}

func moveFileToDestination(srcFile, dstFile string) string {
	err := os.Rename(srcFile, dstFile)

	if err != nil {
		logger("[ERROR]", fmt.Sprintf("Falha ao mover o arquivo: %s. Erro: %s.", srcFile, err.Error()))
		return "ERROR"
	} else {
		logger("[INFO]", fmt.Sprintf("Sucesso ao mover o arquivo: %s.", srcFile))
		return "SUCCESS"
	}
}

func getConfSections() (*configparser.Configuration, []string) {
	configparser.Delimiter = "="

	config, err := configparser.Read(configFile)
	if err != nil {
		logger("[ERROR]", fmt.Sprintf("Falha ao ler arquivo de configuracao: %s. Erro: %s.", configFile, err.Error()))
	}

	sections, err := config.AllSections()

	if err != nil {
		logger("[ERROR]", fmt.Sprintf("Não foi possível ler as Secoes do arquivo de configuracao. Erro: %s.", err.Error()))
	} else {
		var availableSectionNames []string
		for _, section := range sections {
			availableSectionNames = append(availableSectionNames, section.Name())
		}
		return config, availableSectionNames
	}
	return nil, nil
}

func moveFileAndAddLineToTabbyLog(files []fs.DirEntry, t *tabby.Tabby, entry, src string) {
	t.AddHeader("DATA", "HORA", "TIPO", "ORIGEM", "DESTINO", "NOME", "TAMANHO", "STATUS")
	for _, file := range files {
		fileInfo, _ := file.Info()

		if strings.HasSuffix(file.Name(), strings.ToUpper("."+entry)) || strings.HasSuffix(file.Name(), strings.ToLower("."+entry)) {
			status := moveFileToDestination(src+file.Name(), dir+"\\"+file.Name())
			t.AddLine(time.Now().Format("2006/01/02"), time.Now().Format("15:04:05"), entry, src, dir, file.Name(), byteCountDecimal(fileInfo.Size()), status)
		}
	}
	t.AddLine("")
	//t.Print()
}

// Converte um número inteiro que representa um tamanho em bytes (como o tamanho de um arquivo) em uma string legível por humanos
func byteCountDecimal(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

/*===================================================================
	MAIN SCRIPT
===================================================================*/

func main() {
	configBox := box.Config{Px: 1, Py: 1, Type: "Bold", TitlePos: "Top", Color: "Cyan"}
	boxNew := box.Box{TopRight: "+", TopLeft: "+", BottomRight: "+", BottomLeft: "+", Horizontal: "-", Vertical: "|", Config: configBox}

	boxNew.Println("moveLooper", "Organizador de arquivos da pasta de Downloads")

	fd, _ := os.OpenFile(tabbyFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer fd.Close()

	w := tabwriter.NewWriter(fd, 0, 0, 4, ' ', 0)
	t := tabby.NewCustom(w)

	config, sectionNames := getConfSections()

	for _, sectionName := range sectionNames {

		if strings.Contains(sectionName, "global") {
			continue
		} else {
			section, err := config.Section(sectionName)

			if err != nil {
				logger("[ERROR]", fmt.Sprintf("Falha ao obter secao. Erro: %s.", err.Error()))
			} else {
				src, dst := section.ValueOf("source"), section.ValueOf("destination")
				entries := strings.Split(section.ValueOf("entries"), ",")

				for _, entry := range entries {

					fmt.Println("Processando", sectionName, "->", "."+entry)
					dir = dst + entry
					createDirectory(dir)

					files, err := os.ReadDir(src)

					if err != nil {
						logger("[ERROR]", fmt.Sprintf("Falha ao ler o diretorio: %s. Erro: %s.", src, err.Error()))
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
						logger("[INFO]", fmt.Sprintf("Nenhum arquivo %s para mover", entry))
						fmt.Printf("┖─── Nenhum arquivo %s para mover\n", entry)
					case 1:
						logger("[INFO]", fmt.Sprintf("%s arquivo %s para mover", strconv.Itoa(countFiles), entry))
						fmt.Printf("┖─── %s arquivo %s para mover\n", strconv.Itoa(countFiles), entry)
						moveFileAndAddLineToTabbyLog(files, t, entry, src)
					default:
						logger("[INFO]", fmt.Sprintf("%s arquivos %s para mover", strconv.Itoa(countFiles), entry))
						fmt.Printf("┖─── %s arquivos %s para mover\n", strconv.Itoa(countFiles), entry)
						moveFileAndAddLineToTabbyLog(files, t, entry, src)
					}
				}
			}
		}
	}
	//time.Sleep(5 * time.Second)

	fmt.Print("Aperte 'Enter' para continuar...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
