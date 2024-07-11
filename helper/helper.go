package helper

import (
	"fmt"
	"movelooper/logging"
	"os"
)

func CreateDirectory(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			logging.Log.Errorf("Falha ao criar diretorio: %s. Erro: %s.", dir, err.Error())
		}

		logging.Log.Infof("Sucesso ao criar diretorio: %s.", dir)
	}
}

func MoveFileToDestination(srcFile, dstFile string) string {
	err := os.Rename(srcFile, dstFile)

	if err != nil {
		logging.Log.Errorf("Falha ao mover o arquivo: %s. Erro: %s.", srcFile, err.Error())
		return "ERROR"
	}

	logging.Log.Infof("Sucesso ao mover o arquivo: %s.", srcFile)
	return "SUCCESS"
}

// Converte um número inteiro que representa um tamanho em bytes (como o tamanho de um arquivo) em uma string legível por humanos
func ByteCountDecimal(b int64) string {
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
