package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func searchSave(srcDir, dstDir string, filenames2 []string, formats []string) {
	filenames := make(map[string]bool)

	for _, name := range filenames2 {
		filenames[name] = true
		filenames[name+".zip"] = true
	}

	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, "Error:", err)
			return nil
		}
		if !info.IsDir() {
			fileName := info.Name()
			ext := filepath.Ext(fileName)
			name := fileName[:len(fileName)-len(ext)]

			if filenames[name] && contains(formats, ext) {
				dstPath := filepath.Join(dstDir, fileName)
				fmt.Printf("Copy %s to %s\n", path, dstPath)
				copyFile(path, dstPath)
			}
		}
		return nil
	})
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func startParsing() {
	srcDir := "\\\\s-bki\\КИ\\Эквифакс"
	dstDir := "C:\\output"

	filenames2 := []string{
		"0OA_FCH_20241118_0018",
		"0OA_FCH_20241127_0008",
		"0OA_FCH_20241114_0030",
	}

	formats := []string{".zip", ".sgn", ".sig"} // Эквифакс 
	// formats := []string{".zip", ".'.pem'", ".pem.pem"} // ОКБ
	// formats := []string{".zip"} // НБКИ

	searchSave(srcDir, dstDir, filenames2, formats)
}
