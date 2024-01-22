package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Main function
// TODO: Add support for multiple pages
func main() {
	pageStart := 10
	pageLast := 12
	listOfStringsToRemove := []string{
		"After you answer a question in this section, you will NOT be able to return to it. As a result, these questions will not appear in the review screen.",
	}

	// Go through pages
	for pageStart <= pageLast {
		res, err := http.Get("https://free-braindumps.com/microsoft/free-az-104-braindumps.html?p=" + fmt.Sprint(pageStart))
		if err != nil {
			fmt.Println(err)
			return
		}
		defer res.Body.Close()

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get string after the column from the first .panel-title as a file name with lowercase and spaces replaced with dashes
		title := doc.Find(".panel-title .pull-right").First().Text()
		splitTitle := strings.Split(title, ":")
		pageTitle := strings.TrimSpace(splitTitle[1])       // Trim spaces
		pageTitle = strings.ToLower(pageTitle)              // Convert to lowercase
		pageTitle = strings.ReplaceAll(pageTitle, " ", "-") // Replace spaces with dashes

		pageString := fmt.Sprintf("%03d", pageStart)
		fileAk, err := os.Create("../az-104/aiken/" + pageString + "-" + pageTitle + ".ak")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer fileAk.Close()
		fileGift, err := os.Create("../az-104/gift/" + pageString + "-" + pageTitle + ".gift")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer fileGift.Close()
		// MoodleXML
		fileMoodleXML, err := os.Create("../az-104/xml/" + pageString + "-" + pageTitle + ".xml")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer fileMoodleXML.Close()
		// Prepend <?xml version="1.0"?> to moodle XML file
		fileMoodleXML.WriteString("<?xml version=\"1.0\"?>\n<quiz>\n")

		doc.Find(".panel").Each(func(i int, s *goquery.Selection) {
			// Get Question number after the column from the first .panel-title > .text-uppercase
			questionNumber := s.Find(".panel-title > .text-uppercase").First().Text()

			splitQuestionNumber := strings.Split(questionNumber, ":")
			if len(splitQuestionNumber) == 2 {
				questionNumber = splitQuestionNumber[1]
				questionNumber = strings.TrimSpace(questionNumber)           // Trim spaces
				questionNumber = strings.ReplaceAll(questionNumber, " ", "") // Remove spaces
				questionNumber = strings.ReplaceAll(questionNumber, ".", "") // Remove dots
			} else {
				questionNumber = ""
			}

			question, _ := s.Find(".lead").First().Html()

			// Check if the question is not empty, if it is, skip it
			if question == "" {
				return
			}

			// Remove the note from the question
			re := regexp.MustCompile(`Note:.*?<br/>`)    // Regular expression to match the note
			question = re.ReplaceAllString(question, "") // Remove the note

			// Remove all strings from the list of strings to remove
			for _, stringToRemove := range listOfStringsToRemove {
				re = regexp.MustCompile(stringToRemove)      // Regular expression to match the string to remove
				question = re.ReplaceAllString(question, "") // Remove the string
			}

			// Remove all extra spaces from the question
			re = regexp.MustCompile(`\s+`)
			question = re.ReplaceAllString(question, " ")
			question = strings.TrimSpace(question)
			question = strings.ReplaceAll(question, "  ", " ")

			// Get the choices and answer
			choices := s.Find("li").Map(func(i int, s *goquery.Selection) string {
				return s.Text()
			})
			answers := s.Find("li[data-correct='True']").Map(func(i int, s *goquery.Selection) string {
				return strconv.Itoa(s.Index())
			})

			// Convert answers from []string to []int
			convertedAnswers := make([]int, len(answers))
			for i, answer := range answers {
				convertedAnswer, _ := strconv.Atoi(answer) // Convert answer from string to int
				convertedAnswers[i] = convertedAnswer
			}

			gift := formatGIFT(questionNumber, question, choices, convertedAnswers)
			writeToFile(fileGift, gift)

			println("Choices: " + fmt.Sprint(choices))
			println("Answers: " + fmt.Sprint(convertedAnswers))

			moodleXML := formatMoodleXML(question, choices, convertedAnswers)
			writeToFile(fileMoodleXML, moodleXML)

			aiken := formatAiken(question, choices, convertedAnswers)
			writeToFile(fileAk, aiken)

		})

		// Append </quiz> to moodle XML file
		fileMoodleXML.WriteString("</quiz>\n")
		pageStart++
	}
}

// Write the content to the file
//
// @param file The file
// @param content The content to write to the file
// @return void
func writeToFile(file *os.File, content string) {
	_, err := file.WriteString(content + "\n")
	if err != nil {
		fmt.Println(err)
		return
	}
}

// Remove all HTML tags from the string
//
// @param str The string
// @return The string without HTML tags
func _removeHtmlTags(str string) string {
	re := regexp.MustCompile(`<.*?>`)   // Regular expression to match HTML tags
	str = re.ReplaceAllString(str, " ") // Remove HTML tags
	return str
}

// Format the question and choices into the Aiken format
//
// @param question The question
// @param choices The choices
// @param answers The answers
// @return The Aiken formatted question
func formatAiken(question string, choices []string, answers []int) string {
	formatted := _removeHtmlTags(question) + "\n"

	// Prefix the choices with A...Z and a dot
	for i, choice := range choices {
		choices[i] = fmt.Sprintf("%c. %s", rune('A'+i), _removeHtmlTags(choice))
		formatted += choice + "\n"
	}
	formatted += "ANSWER: "
	for i, answer := range answers {
		if i > 0 {
			formatted += ", "
		}
		formatted += string(rune('A' + answer))
	}
	return formatted + "\n"
}

// Format the question and choices into the GIFT format
//
// @param questionNumber The question number
// @param question The question
// @param choices The choices
// @param answers The answers
// @return The GIFT formatted question
func formatGIFT(questionNumber string, question string, choices []string, answers []int) string {
	formatted := "::Q" + questionNumber + " " + _removeHtmlTags(question) + "::" + "\n{\n"
	for i, choice := range choices {
		if i > 0 {
			formatted += "\n~"
		} else {
			formatted += " "
		}
		if contains(answers, i) {
			formatted += "="
		}
		formatted += _removeHtmlTags(choice)
	}
	formatted += "\n}\n"
	return formatted
}

// Check if a slice contains an item
//
// @param slice The slice
// @param item The item
// @return True if the slice contains the item, false otherwise
func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// Format the question and choices into the Moodle XML format
//
// @param question The question
// @param choices The choices
// @param answers The answers
// @return The Moodle XML formatted question
func formatMoodleXML(question string, choices []string, answers []int) string {
	formatted := "<question type=\"multichoice\">\n"
	formatted += "<name><text>" + _removeHtmlTags(question) + "</text></name>\n"
	formatted += "<questiontext format=\"html\"><text><![CDATA[" + question + "]]></text></questiontext>\n"
	formatted += "<answernumbering>abc</answernumbering>\n"
	for i, choice := range choices {
		if contains(answers, i) {
			formatted += "<answer fraction=\"100\">\n"
		} else {
			formatted += "<answer fraction=\"0\">\n"
		}
		formatted += "<text>" + choice + "</text>\n"
		formatted += "</answer>\n"
	}
	formatted += "</question>\n"
	return formatted
}
