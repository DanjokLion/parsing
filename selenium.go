package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

var (
	chromeDriverPath = "./chromedriver.exe"
	downloadDir      = "C:\\запрос_цб_много\\НБКИ\\тикеты"
	loginURL         = "https://lk.nbki.ru/Cabinet/login/auth"
	username         string
	password         string
)

func isFileDownloaded(fileName string) bool {
	expectedFile := fileName + "_ticket1.ZIP.ENC"
	path := filepath.Join(downloadDir, expectedFile)
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func downloadFile(wd selenium.WebDriver, fileName string) bool {
	rows, err := wd.FindElements(selenium.ByXPATH, "//tr[contains(@class, 'odd') or contains(@class, 'even')]")
	if err != nil {
		log.Println("Error finding rows:", err)
		return false
	}

	for _, row := range rows {
		link, err := row.FindElement(selenium.ByXPATH, ".//td[2]/a")
		if err != nil {
			continue
		}

		text, err := link.Text()
		if err != nil {
			continue
		}

		fileText := strings.TrimSpace(strings.Replace(text, ".ZIP.ENC", "", 1))
		if fileText == fileName && !isFileDownloaded(fileText) {
			fmt.Printf("Found file: %s\n", fileText)
			link.Click()
			time.Sleep(1 * time.Second)

			if handleDownload(wd, fileName) {
				return true
			}
			
			wd.Back()
			time.Sleep(1 * time.Second)
		}
	}
	return false
}

func handleDownload(wd selenium.WebDriver, fileName string) bool {
	ticket1, err := waitAndFind(wd, selenium.ByXPATH, "//a[contains(@href, 'ticket1')]", 10)
	if err != nil {
		log.Println("Error finding ticket1:", err)
		return false
	}

	ticket2, err := waitAndFind(wd, selenium.ByXPATH, "//a[contains(@href, 'ticket2')]", 10)
	if err != nil {
		log.Println("Error finding ticket2:", err)
		return false
	}

	ticket1URL, _ := ticket1.GetAttribute("href")
	ticket2URL, _ := ticket2.GetAttribute("href")

	for _, url := range []string{ticket1URL, ticket2URL} {
		if err := wd.Get(url); err != nil {
			log.Println("Error downloading ticket:", err)
			return false
		}
		time.Sleep(2 * time.Second)
	}

	if isFileDownloaded(fileName) {
		fmt.Printf("Successfully downloaded: %s\n", fileName)
		return true
	}
	
	fmt.Printf("Download failed: %s\n", fileName)
	return false
}

func processFiles(wd selenium.WebDriver, fileList []string) {
	for _, fileName := range fileList {
		if isFileDownloaded(fileName) {
			fmt.Printf("File already exists: %s\n", fileName)
			continue
		}

		fmt.Printf("\n=== Processing file: %s ===\n", fileName)
		currentPage := 1

		for {
			fmt.Printf("Checking page %d for %s\n", currentPage, fileName)
			if downloadFile(wd, fileName) {
				break
			}

			nextBtn, err := waitAndFind(wd, selenium.ByXPATH, "//a[@class='nextLink']", 5)
			if err != nil {
				fmt.Println("Reached end of pages")
				break
			}

			nextBtn.Click()
			currentPage++
			time.Sleep(2 * time.Second)
		}

		wd.Back()
	}
}

func waitAndFind(wd selenium.WebDriver, by, value string, timeout time.Duration) (selenium.WebElement, error) {
	var elem selenium.WebElement
	var err error
	start := time.Now()

	for time.Since(start) < timeout*time.Second {
		elem, err = wd.FindElement(by, value)
		if err == nil {
			return elem, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, err
}

func startSelenium() {

	username := os.Getenv("USERNAME_NBKI")
	password := os.Getenv("PASSWORD_NBKI")

	service, err := selenium.NewChromeDriverService(chromeDriverPath, 4444)
	if err != nil {
		log.Fatal("Error starting driver:", err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{
		"browserName": "chrome",
	}

	chromeCaps := chrome.Capabilities{
		Prefs: map[string]interface{}{
			"download.default_directory": downloadDir,
			"download.prompt_for_download": false,
			"directory_upgrade":            true,
			"safebrowsing.enabled":         true,
		},
	}
	caps.AddChrome(chromeCaps)

	wd, err := selenium.NewRemote(caps, "")
	if err != nil {
		log.Fatal("Error connecting to WebDriver:", err)
	}
	defer wd.Quit()

	if err := wd.Get(loginURL); err != nil {
		log.Fatal("Error loading login page:", err)
	}

	loginField, _ := waitAndFind(wd, selenium.ByName, "j_username", 10)
	passField, _ := waitAndFind(wd, selenium.ByName, "j_password", 10)
	submitBtn, _ := waitAndFind(wd, selenium.ByID, "submit", 10)

	loginField.SendKeys(username)
	passField.SendKeys(password)
	submitBtn.Click()

	newsLink, _ := waitAndFind(wd, selenium.ByXPATH, "//*[contains(text(), 'Новости')]", 10)
	newsLink.Click()

	gutdfLink, _ := waitAndFind(wd, selenium.ByXPATH, "//a[contains(text(), 'Результаты обработки RUTDF-файлов')]", 10)
	gutdfLink.Click()

	_, err = waitAndFind(wd, selenium.ByXPATH, "//*[contains(text(), 'RUTDF файлы')]", 10)
	if err != nil {
		log.Fatal("Error navigating to GUTDF section:", err)
	}

	fileList := []string{

	"WD01BB000002_20230731_093247",
	"WD01BB000002_20230727_180008",

	}

	processFiles(wd, fileList)
}