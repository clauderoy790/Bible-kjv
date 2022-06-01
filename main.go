package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

var fetchFrom = "https://www.kjvbibles.com/blogs"
var initial []*Book
var enhanced []*BookEnhanced
var enhancedVerses = make(map[string]map[int]map[int]*VerseEnhanced)
var urlExceptions = map[string]string{
	"1-samuel-chapter-16": "1-samuel-chapter-166",
}
var verseTitlesExceptions = map[string]map[string]map[string]string{}
var enhancements []Enhancement
var logFile = "./logs.txt"
var initialPath = "./json/initial"
var enhancedPath = "./json/enhanced"
var cachePath = "./cache"

func main() {
	loadInitialBooks()
	fetchBibleData()
	// makeExceptionEnhancements()
	// applyEnhancements()
	// writeEnhancedBooks()
}

func getEnhancedBook(title string) *BookEnhanced {
	for _, b := range enhanced {
		if b.Title == title {
			return b
		}
	}
	return nil
}

func makeExceptionEnhancements() {
	fmt.Println("TODO")
	for book, chapters := range verseTitlesExceptions {
		for chapter, verses := range chapters {
			fmt.Println(verses)
			fmt.Println(chapter)
			fmt.Println(book)
		}
	}
}

func loadInitialBooks() {
	files, err := ioutil.ReadDir(initialPath)
	if err != nil {
		panic(err)
	}
	fmt.Println("reading initial data...")
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		bytes, err := ioutil.ReadFile(initialPath + "/" + file.Name())
		if err != nil {
			log.Fatal(err)
		}
		book := new(Book)
		if err = json.Unmarshal(bytes, book); err == nil {
			enhancedBook := convertBookToEnhanced(book)
			enhanced = append(enhanced, enhancedBook)
			for _, c := range enhancedBook.Chapters {
				c := c
				for _, v := range c.Verses {
					v := v
					if _, ok := enhancedVerses[enhancedBook.Title]; !ok {
						enhancedVerses[enhancedBook.Title] = make(map[int]map[int]*VerseEnhanced)
					}
					if _, ok := enhancedVerses[enhancedBook.Title][c.Nb]; !ok {
						enhancedVerses[enhancedBook.Title][c.Nb] = make(map[int]*VerseEnhanced)
					}
					enhancedVerses[enhancedBook.Title][c.Nb][v.Nb] = v
				}
			}
			initial = append(initial, book)
		}
	}
}

func convertBookToEnhanced(book *Book) *BookEnhanced {
	var chaptersEn []*ChapterEnhanced
	for _, chap := range book.Chapters {
		chapEn := new(ChapterEnhanced)
		nb, err := strconv.Atoi(chap.Chapter)
		if err != nil {
			panic(fmt.Errorf("faield to convert chapter.chapter to int: %s", chap.Chapter))
		}
		var versesEn []*VerseEnhanced
		for _, verse := range chap.Verses {
			verseEn := new(VerseEnhanced)
			nb, err := strconv.Atoi(verse.Verse)
			if err != nil {
				panic(fmt.Errorf("failed to convert verse to int: %s", verse.Verse))
			}
			verseEn.Nb = nb
			verseEn.Text = verse.Text
			if verse.Title != "" {
				verseEn.Title = verse.Title
			}
			versesEn = append(versesEn, verseEn)
		}
		chapEn.Verses = versesEn
		chapEn.Nb = nb
		chaptersEn = append(chaptersEn, chapEn)
	}
	return &BookEnhanced{
		Title:    book.Book,
		Chapters: chaptersEn,
	}
}

func countTotalToFetch() int {
	nb := 0
	for _, b := range initial {
		for _, _ = range b.Chapters {
			nb++
		}
	}
	return nb
}

func fetchBibleData() {
	fmt.Println("fetching/processing data...")
	for _, book := range initial {
		for _, chapter := range book.Chapters {
			fullURL := getFullUrl(book, chapter)
			if _, err := os.Stat(getCacheFileName(book, chapter)); err == nil {
				if cacheBytes, err := ioutil.ReadFile(getCacheFileName(book, chapter)); err == nil {
					tryWriteEnhancements(book, chapter, string(cacheBytes))
				}
				continue
			}
			fmt.Printf("Fetching: %s - Chapter %s\n", book.Book, chapter.Chapter)

			bodyStr, err := scrapeURL(fullURL)
			if err != nil {
				logError(fmt.Errorf("failed to scrape: %s, error: %w", fullURL, err))
			}
			tryWriteEnhancements(book, chapter, bodyStr)
			cacheData(book, chapter, []byte(bodyStr))
			time.Sleep(time.Second * 2)
		}
	}
}

func scrapeURL(url string) (res string, err error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			res, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	)

	logError(fmt.Errorf("error scraping %s: %w", url, err))

	return res, nil
}

func getFullUrl(book *Book, chapter *Chapter) string {
	bookPath := strings.ReplaceAll(strings.ToLower(book.Book+"/"), " ", "-")
	chapterPath := strings.ToLower(fmt.Sprintf("%s-chapter-%s", strings.ReplaceAll(book.Book, " ", "-"), chapter.Chapter))
	if book.Book == "1-samuel" && chapter.Chapter == "16" {
		fmt.Println("salut: ")
	}
	// some URL don't have the one that they should so replace with exception
	if val, ok := urlExceptions[chapterPath]; ok {
		chapterPath = val
	}
	fullURL := strings.ToLower(fetchFrom + "/" + bookPath + chapterPath)
	fullURL = strings.TrimSpace(strings.ReplaceAll(fullURL, " ", "-"))
	return fullURL
}

func logError(err error) {
	fmt.Println(err)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		_ = ioutil.WriteFile(logFile, []byte(""), 0644)
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		log.Println("error opening log file:", err)
		return
	}
	if _, err := file.WriteString(err.Error()); err != nil {
		log.Println("error writing to log file:", err)
	}
}

func tryWriteEnhancements(book *Book, chapter *Chapter, htmlData string) {
	display := fmt.Sprintf("%s - %s", book.Book, chapter.Chapter)
	// todo here
	// if !(book.Book == "Genesis" && chapter.Chapter == "1") {
	// 	return
	// }
	fmt.Printf("Writing enchancements for %s - %s \n\n", book.Book, chapter.Chapter)
	document, err := goquery.NewDocumentFromReader(strings.NewReader(htmlData))
	if err != nil {
		fmt.Println("failed to create document")
		panic(err)
	}

	if err := titleIsExpected(document, book, chapter); err != nil {
		logError(err)
		panic(err)
	}

	parSelector := ".post p"
	containers := document.Find(parSelector)
	if containers == nil {
		panic(fmt.Errorf("fail to find paragraphs for %s - %v", book.Book, chapter.Chapter))
	}

	if book.Book == "1 Chronicles" && chapter.Chapter == "1" {
		fmt.Println("debug")
	}

	currT := ""
	curVerse := 0
	startVerse := -1
	containers.Each(func(i int, par *goquery.Selection) {
		title := ""
		text := ""
		if currT == "" {
			strong := par.Find("strong").First()
			if strong == nil {
				panic("unable to find strong")
			}
			// title
			title = strings.TrimSpace(strong.Text())
			title = strings.TrimSuffix(title, ".")
		} else {
			title = currT
		}

		// text
		text = par.Text()
		text = strings.ReplaceAll(text, "\u2009", " ") // replace thin spaces by spaces
		text = strings.TrimSpace(text)
		currT = ""
		if text == title {
			currT = title
			return
		}

		if strings.HasPrefix(text, title) {
			text = strings.TrimSpace(strings.Replace(text, title, "", 1))
		}

		// get chapter text
		for text != "" {
			startText := verseStartText(curVerse + 1)
			curVerse++
			if startVerse == -1 {
				startVerse = curVerse
			}
			if curVerse == 17 {
				fmt.Println("look")
			}

			if strings.HasPrefix(text, startText) {
				if i := strings.Index(text, verseStartText(curVerse+1)); i != -1 {
					text = strings.Replace(text, startText, "", 1)
					i = strings.Index(text, verseStartText(curVerse+1))
					actualText := strings.TrimSpace(text[:i])
					setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actualText)
					text = strings.TrimSpace(text[i:])

				} else {
					// at end of chapter
					text = strings.Replace(text, startText, "", 1)
					actual := strings.TrimSpace(text)
					setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)

					text = strings.Replace(text, actual, "", 1)
				}
			} else {
				// at end of paragraph
				text = strings.Replace(text, startText, "", 1)
				actual := strings.TrimSpace(text)
				setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)
				text = strings.Replace(text, actual, "", 1)
			}
		}

		// title per verse
		chapNb, _ := strconv.Atoi(chapter.Chapter)
		en := Enhancement{
			book:    book.Book,
			chapter: chapNb,
			verse:   startVerse,
			title:   title,
		}
		enhancements = append(enhancements, en)
		startVerse = -1

		fmt.Printf("Created new enhancements for %s - %s\n%+v\n", book.Book, chapter.Chapter, en)
	})

	// verify if verse count is the same
	if len(chapter.Verses) != curVerse {
		panic(fmt.Errorf("chapter %s doesnt have the same amount of verse, original:%v, actual:%v", display, len(chapter.Verses), curVerse))
	}
	// panic("LETS GOOOOO!")
}

func verseStartText(verse int) string {
	return fmt.Sprintf("%d ", verse)
}

func setEnhancedVerseText(book, chapter, verse, text string) {
	var v *VerseEnhanced
	eb := getEnhancedBook(book)
	cNb, _ := strconv.Atoi(chapter)
	vNb, _ := strconv.Atoi(verse)
	for _, c := range eb.Chapters {
		if c.Nb == cNb {
			for _, ver := range c.Verses {
				if ver.Nb == vNb {
					v = ver
					break
				}
			}
		}
	}
	if v == nil {
		panic(fmt.Errorf("failed to find en: %s", book+"-"+chapter+"-"+verse))
	}
	v.Text = text
}

func isNewVerse(currentChapter int, text string) bool {
	return strings.HasPrefix(text, fmt.Sprintf("%d ", (currentChapter+1)))
}

func titleIsExpected(document *goquery.Document, book *Book, chapter *Chapter) error {
	pageTitleElement := document.Find("title")
	if pageTitleElement == nil {
		return fmt.Errorf("could not find page title")
	}
	pageTitle := pageTitleElement.Text()
	pageTitle = strings.ReplaceAll(pageTitle, "– KJV Bibles", "")
	pageTitle = strings.TrimSpace(pageTitle)
	expectedTitle := strings.TrimSpace(fmt.Sprintf("%s Chapter %v", book.Book, chapter.Chapter))
	if !strings.EqualFold(pageTitle, expectedTitle) {
		fmt.Println("page title: ", pageTitle)
		fmt.Println("expected title: ", expectedTitle)
		errorMessage := "check Website, page title doesn't match current book/chapter: " + pageTitle + ", expecting: " + expectedTitle
		return fmt.Errorf(errorMessage)
	}
	return nil
}

func verifyTitle() bool {
	return false
}

func isGenesis1(book *Book, chapter *Chapter) bool {
	return book.Book == "Genesis" && chapter.Chapter == "1"
}

func cacheData(book *Book, chapter *Chapter, bytes []byte) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		_ = os.MkdirAll(cachePath, 0777)
	}
	fileName := getCacheFileName(book, chapter)
	err := ioutil.WriteFile(fileName, bytes, 0777)
	if err != nil {
		log.Fatalf("fail to cache file: %s\n", fileName)
	}
}

func getCacheFileName(book *Book, chapter *Chapter) string {
	wd, _ := os.Getwd()
	return path.Join(wd, cachePath, strings.ReplaceAll(strings.ToLower(fmt.Sprintf("%s-%s.html", book.Book, chapter.Chapter)), " ", ""))
}

func applyEnhancements() {
	for _, en := range enhancements {
		verse := enhancedVerses[en.book][en.chapter][en.verse]
		verse.Title = en.title
		fmt.Printf("set new verse: %s - %d - %d: %s\n", en.book, en.chapter, en.verse, verse.Title)
	}
	fmt.Printf("applied %v enhancements!\n", len(enhancements))
}

func writeEnhancedBooks() {
	os.RemoveAll(enhancedPath)
	if err := os.MkdirAll(enhancedPath, 0777); err != nil {
		panic(err)
	}

	wroteCount := 0
	for _, book := range enhanced {
		bytes, err := json.Marshal(book)
		if err != nil {
			panic(err)
		}
		wroteCount++
		fileName := fmt.Sprintf("%s/%s.json", enhancedPath, strings.ReplaceAll(book.Title, " ", ""))
		if err := ioutil.WriteFile(fileName, bytes, 0777); err == nil {
			fmt.Println("wrote file: ", fileName)
		}
	}
	fmt.Printf("wrote a total of %v books!\n", wroteCount)

}

func deepClone(books []*Book) []*Book {
	bytes, err := json.Marshal(books)
	if err != nil {
		panic(fmt.Errorf("error marshalign for deep copy: %w", err))
	}
	clone := make([]*Book, 0)
	if err := json.Unmarshal(bytes, &clone); err != nil {
		panic(fmt.Errorf("error unmarshanlig for ddep copy: %w", err))
	}
	return clone
}

type Enhancement struct {
	title   string
	verse   int
	chapter int
	book    string
}

type Book struct {
	Book     string     `json:"book"`
	Chapters []*Chapter `json:"chapters"`
}

type Chapter struct {
	Chapter string   `json:"chapter"`
	Verses  []*Verse `json:"verses"`
}

type Verse struct {
	Title string `json:"title,omitempty"`
	Verse string `json:"verse"`
	Text  string `json:"text"`
}

type BookEnhanced struct {
	Title    string             `json:"title,omitempty"`
	Chapters []*ChapterEnhanced `json:"chapters"`
}

type ChapterEnhanced struct {
	Nb     int              `json:"nb"`
	Verses []*VerseEnhanced `json:"verses"`
}

type VerseEnhanced struct {
	Title string `json:"title,omitempty"`
	Nb    int    `json:"nb"`
	Text  string `json:"text"`
}
