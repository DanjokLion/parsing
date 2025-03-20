package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

var (
	imapServer         string
	emailUser          string
	emailPass          string
	targetDir          = `C:\Users\popov.d\Desktop\parse\parser_mail_go\test_papka`
	subjectPrefix      = "[okb] "
	batchSize          = 20
	fetchBatchSize     = 100
	maxReconnectAttempts = 3
)

// сюда прокидываем файлы, которые нужно найти
var fileList = [...]string{
	"CHP_01465_UCH_03-00_20250124022222357",
}

func startEmail() {

	imapServer = os.Getenv("IMAP_SERVER")
	emailUser = os.Getenv("EMAIL_USER")
	emailPass = os.Getenv("EMAIL_PASS")
	
	start := time.Now()
	log.SetOutput(os.Stdout)
	log.Printf("Старт поиска для %d файлов", len(fileList))

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("Не удалось создать директорию: %v", err)
	}

	c := connectIMAP()
	defer c.Logout()

	mbox, err := c.Select("INBOX", true)
	if err != nil {
		log.Fatalf("Ошибка выбора папки: %v", err)
	}
	log.Printf("Выбрана папка: INBOX (Всего писем: %d)", mbox.Messages)

	foundUIDs := make(map[uint32]struct{})
	for i := 0; i < len(fileList); i += batchSize {
		end := i + batchSize
		if end > len(fileList) {
			end = len(fileList)
		}
		batch := fileList[i:end]
		
		uids := searchBatch(c, batch)
		for _, uid := range uids {
			foundUIDs[uid] = struct{}{}
		}
	}

	if len(foundUIDs) == 0 {
		log.Fatal("Совпадений не найдено")
	}

	uidSeq := make([]uint32, 0, len(foundUIDs))
	for uid := range foundUIDs {
		uidSeq = append(uidSeq, uid)
	}

	log.Printf("Найдено %d уникальных писем, начинаем загрузку...", len(uidSeq))
	messages := fetchMessagesWithRetry(c, uidSeq)
	saveAttachments(messages)

	log.Printf("Обработка завершена за %v", time.Since(start))
}

func connectIMAP() *client.Client {
	log.Printf("Подключаемся к %s...", imapServer)
	c, err := client.DialTLS(imapServer, nil)
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}

	log.Printf("Авторизуемся как %s...", emailUser)
	if err := c.Login(emailUser, emailPass); err != nil {
		log.Fatalf("Ошибка авторизации: %v", err)
	}
	
	return c
}

func reconnectIMAP() *client.Client {
	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		log.Printf("Попытка переподключения %d/%d...", attempt, maxReconnectAttempts)
		
		c, err := client.DialTLS(imapServer, nil)
		if err != nil {
			log.Printf("Ошибка подключения: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		if err := c.Login(emailUser, emailPass); err != nil {
			log.Printf("Ошибка авторизации: %v", err)
			c.Logout()
			time.Sleep(2 * time.Second)
			continue
		}
		
		log.Println("Переподключение успешно")
		return c
	}
	
	log.Printf("Не удалось переподключиться после %d попыток", maxReconnectAttempts)
	return nil
}

func searchBatch(c *client.Client, batch []string) []uint32 {
	allUIDs := make(map[uint32]struct{})
	
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Subject", subjectPrefix)
	
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, 0) 
	
	prefixUIDs, err := c.Search(criteria)
	if err != nil {
		log.Printf("Ошибка поиска по префиксу: %v", err)
		return nil
	}
	
	log.Printf("Найдено писем с префиксом [%s]: %d", subjectPrefix, len(prefixUIDs))
	
	if len(prefixUIDs) == 0 {
		return nil
	}
	
	var sortedUIDs []uint32
	
	for i := 0; i < len(prefixUIDs); i += fetchBatchSize {
		end := i + fetchBatchSize
		if end > len(prefixUIDs) {
			end = len(prefixUIDs)
		}
		
		batchUIDs := prefixUIDs[i:end]
		seqSet := new(imap.SeqSet)
		seqSet.AddNum(batchUIDs...)
		
		items := []imap.FetchItem{imap.FetchUid, imap.FetchInternalDate}
		messages := make(chan *imap.Message, 10)
		
		done := make(chan error, 1)
		go func() {
			done <- c.Fetch(seqSet, items, messages)
		}()
		
		type uidDate struct {
			uid  uint32
			date time.Time
		}
		
		var dateList []uidDate
		for msg := range messages {
			dateList = append(dateList, uidDate{
				uid:  msg.Uid,
				date: msg.InternalDate,
			})
		}
		
		if err := <-done; err != nil {
			log.Printf("Ошибка получения дат писем: %v", err)
			continue
		}
		
		for i := 0; i < len(dateList); i++ {
			for j := i + 1; j < len(dateList); j++ {
				if dateList[i].date.Before(dateList[j].date) {
					dateList[i], dateList[j] = dateList[j], dateList[i]
				}
			}
		}
		
		for _, item := range dateList {
			sortedUIDs = append(sortedUIDs, item.uid)
		}
	}
	
	log.Printf("Сортировка завершена, обрабатываем письма от новых к старым")
	
	for i := 0; i < len(sortedUIDs); i += fetchBatchSize {
		end := i + fetchBatchSize
		if end > len(sortedUIDs) {
			end = len(sortedUIDs)
		}
		
		batchUIDs := sortedUIDs[i:end]
		log.Printf("Обрабатываем пакет писем %d-%d из %d", i+1, end, len(sortedUIDs))
		
		seqSet := new(imap.SeqSet)
		seqSet.AddNum(batchUIDs...)
		
		items := []imap.FetchItem{imap.FetchEnvelope}
		messages := make(chan *imap.Message, 10)
		
		done := make(chan error, 1)
		go func() {
			done <- c.Fetch(seqSet, items, messages)
		}()
		
		for msg := range messages {
			subject := msg.Envelope.Subject
			
			for _, filename := range batch {
				fullName := filename + ".xml"
				
				if strings.Contains(subject, "Извещение о получении файла "+fullName) ||
				   strings.Contains(subject, "Извещение о принятии кредитной информации из файла "+fullName) ||
				   strings.Contains(subject, "Извещение о непринятии кредитной информации из файла "+fullName) {
					allUIDs[msg.Uid] = struct{}{}
					log.Printf("Найдено соответствие для файла %s: UID %d", filename, msg.Uid)
					break
				}
			}
		}
		
		if err := <-done; err != nil {
			log.Printf("Ошибка получения пакета конвертов: %v", err)
			
			time.Sleep(2 * time.Second)
			
			if strings.Contains(err.Error(), "connection closed") {
				log.Println("Переподключение к серверу...")
				newClient := reconnectIMAP()
				if newClient != nil {
					c = newClient
					_, err := c.Select("INBOX", true)
					if err != nil {
						log.Printf("Ошибка выбора ящика после переподключения: %v", err)
						break
					}
				} else {
					log.Println("Не удалось переподключиться, пропускаем оставшиеся письма")
					break
				}
			}
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	result := make([]uint32, 0, len(allUIDs))
	for uid := range allUIDs {
		result = append(result, uid)
	}
	
	log.Printf("Найдено совпадений в текущем пакете: %d", len(result))
	return result
}

func fetchMessagesWithRetry(c *client.Client, uids []uint32) chan *imap.Message {
	if len(uids) == 0 {
		log.Println("Нет писем для загрузки")
		return nil
	}
	
	const messageBatchSize = 10
	messages := make(chan *imap.Message, 100)
	
	go func() {
		defer close(messages)
		
		for i := 0; i < len(uids); i += messageBatchSize {
			end := i + messageBatchSize
			if end > len(uids) {
				end = len(uids)
			}
			batchUIDs := uids[i:end]
			
			log.Printf("Загрузка писем %d-%d из %d", i+1, end, len(uids))
			
			seqSet := new(imap.SeqSet)
			seqSet.AddNum(batchUIDs...)
			
			section := &imap.BodySectionName{
				Peek: true,
			}
			
			items := []imap.FetchItem{
				imap.FetchEnvelope,
				imap.FetchFlags,
				imap.FetchInternalDate,
				imap.FetchBodyStructure,
				section.FetchItem(),
			}
			
			batchMessages := make(chan *imap.Message, 10)
			done := make(chan error, 1)
			
			go func() {
				done <- c.UidFetch(seqSet, items, batchMessages)
			}()
			
			for msg := range batchMessages {
				messages <- msg
			}
			
			if err := <-done; err != nil {
				log.Printf("Ошибка при загрузке писем: %v, повторная попытка", err)
				
				time.Sleep(2 * time.Second)
				
				if strings.Contains(err.Error(), "connection closed") {
					log.Println("Переподключение к серверу...")
					newClient := reconnectIMAP()
					if newClient != nil {
						c = newClient
						_, err := c.Select("INBOX", true)
						if err == nil {
							i -= messageBatchSize // Возвращаемся назад для повтора
							time.Sleep(1 * time.Second)
							continue
						} else {
							log.Printf("Ошибка выбора ящика после переподключения: %v", err)
						}
					} else {
						log.Println("Не удалось переподключиться, пропускаем оставшиеся письма")
					}
				}
			}
			
			time.Sleep(500 * time.Millisecond)
		}
	}()
	
	return messages
}

func saveAttachments(messages chan *imap.Message) {
	if messages == nil {
		return
	}
	
	count := 0
	for msg := range messages {
		savedFiles := processMessage(msg)
		count += savedFiles
	}
	log.Printf("Сохранено вложений: %d", count)
}

func processMessage(msg *imap.Message) int {
	subject := msg.Envelope.Subject
	log.Printf("Обработка письма [UID:%d]: %s", msg.Uid, subject)

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		log.Printf("Пропускаем письмо без тела (UID:%d)", msg.Uid)
		return 0
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		log.Printf("Ошибка создания ридера для письма [UID:%d]: %v", msg.Uid, err)
		return 0
	}
	defer mr.Close()

	savedCount := 0
	
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Ошибка чтения части письма [UID:%d]: %v", msg.Uid, err)
			continue
		}

		switch h := p.Header.(type) {
		case *mail.AttachmentHeader:
			filename, err := h.Filename()
			if err != nil || filename == "" {
				continue
			}

			data, err := io.ReadAll(p.Body)
			if err != nil {
				log.Printf("Ошибка чтения вложения [%s] из письма [UID:%d]: %v", filename, msg.Uid, err)
				continue
			}

			if saveAttachment(filename, data) {
				savedCount++
			}
		}
	}
	
	if savedCount == 0 {
		log.Printf("В письме [UID:%d] не найдено вложений для сохранения", msg.Uid)
	} else {
		log.Printf("Из письма [UID:%d] сохранено %d вложений", msg.Uid, savedCount)
	}
	
	return savedCount
}

func saveAttachment(filename string, data []byte) bool {
	cleanFilename := strings.ReplaceAll(filename, ":", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "/", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "\\", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "*", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "?", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "\"", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "<", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, ">", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "|", "_")
	
	path := filepath.Join(targetDir, cleanFilename)
	
	if info, err := os.Stat(path); err == nil {
		if info.Size() == int64(len(data)) {
			log.Printf("Файл уже существует (размер совпадает): %s", cleanFilename)
			return false
		}
		
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		cleanFilename = strings.TrimSuffix(cleanFilename, filepath.Ext(cleanFilename)) + 
					  "-" + timestamp + filepath.Ext(cleanFilename)
		path = filepath.Join(targetDir, cleanFilename)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Ошибка сохранения %s: %v", cleanFilename, err)
		return false
	}
	
	log.Printf("Сохранен файл: %s (%d KB)", cleanFilename, len(data)/1024)
	return true
}