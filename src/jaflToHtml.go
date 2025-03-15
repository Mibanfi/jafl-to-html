package main

import (
	"bufio"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/gen2brain/go-fitz"
)

// --- CONSTANTS ---

const DEFAULT_DIR = "."
const DEFAULT_OUTPUT = "output.html"
const DESIRED_EXT = ".xml"
const IMAGE_EXT = ".JPG"

const ANY = -1

const (
	ERR_BAD_CLOSE = "Error: <%s> is being closed, but wasn't the last element opened"
)

const FORMAT_ATTACHMENT =
`
<img src="%s" id="%s"></img>

`

const HEAD =
`<head>
	<link rel="stylesheet" href="flands.css">
	<link rel="stylesheet" href="font-settings.css">
</head>`

const COVER = "Cover.jpg"

const SHEET_ADDRESS = "http://www.sparkfurnace.com/wp-content/media/Adventure-Sheets-FL%s.pdf"
const SHEET1_NAME = "sheet1.jpg"
const SHEET2_NAME = "sheet2.jpg"
const SHEET_PDF_NAME = "Sheet.pdf"

const MENU =
`<div class="menu">
	<table>
		<tr>
			<th colspan="2">Menu</th>
			<td colspan="1"><a href="#sheet1">Sheet 1</a></td>
			<td colspan="1"><a href="#sheet2">Sheet 2</a></td>
		</tr>
	</table>
</div>
`

// --- TYPES ---

type element struct {
	Name string
	Content string
	Attributes map[string]string
}

type stack []element

// --- FLAGS ---
var outputName *string
var verbose *bool
var cover *bool
var skipIntro *bool
var addSheet *bool

// --- MAIN BODY ---

func main() {
	// Assign flags
	outputName = flag.String("o", DEFAULT_OUTPUT, "Filepath to save output in")
	verbose = flag.Bool("v", false, "Enable verbose output")
	cover = flag.Bool("c", false, "Look for a cover file")
	skipIntro = flag.Bool("i", false, "Skip the introductory pages")
	addSheet = flag.Bool("s", false, "Implement a character sheet into the document")
	flag.Parse()

	// Select the directory
	var dir string
	dir = flag.Arg(0)
	if dir == "" {
		dir = DEFAULT_DIR
		fmt.Print("No directory specified. Operating in the current directory.\n")
	}

	// Create a sorted slice
	var filenames []string
	readDir, readErr := os.ReadDir(dir)
	check(readErr)
	for _, f :=  range readDir {
		filenames = append(filenames, f.Name())
	}
	slices.SortFunc(filenames, betterSort)

	// Create an output file
	if *outputName == DEFAULT_OUTPUT {
		fmt.Printf("No output file specified. Output will be saved in %s.\n", DEFAULT_OUTPUT)
	}
	output, errCreate := os.Create(*outputName)
	check(errCreate)

	// Setup content slice and add head
	var content []byte
	content = []byte(HEAD)

	// Add cover page
	if *cover {
		fmt.Print("Importing cover page... ")
		content = slices.Concat(content, []byte(fmt.Sprintf(FORMAT_ATTACHMENT, filepath.Join(dir, COVER), stripExt(COVER))))
		fmt.Println("Done!")
	}

	// Import Adventurers.xml
	fmt.Printf("Processing file %s... ", ADVENTURERS)
	updateStats(filepath.Join(dir, ADVENTURERS))
	fmt.Println("loaded starting classes")

	// Process all files
	for _, fn := range filenames {
		fmt.Printf("Processing file %s... ", fn)
		if strings.Contains(fn, "temp") || strings.Contains(fn, "old") {
			fmt.Println("ignored")
			continue
		}
		if _, e := strconv.Atoi(strings.TrimSuffix(fn, filepath.Ext(fn))); e != nil && *skipIntro {
			fmt.Println("ignored as per skipIntro flag")
			continue
		}
		if fn == ADVENTURERS {
			continue
		}
		fn = filepath.Join(dir, fn)
		switch filepath.Ext(fn) {
			case IMAGE_EXT:
				content = slices.Concat(content, []byte(fmt.Sprintf(FORMAT_ATTACHMENT, fn, stripExt(fn))))
				fmt.Println("imported as image")
			case DESIRED_EXT:
				page, errParse := parse(fn)
				check(errParse)
				if *addSheet {
					content = slices.Concat(content, []byte(MENU))
				}
				content = slices.Concat(content, []byte(page))
				fmt.Println("done!")
			default:
				fmt.Println("ignored")
		}
	}

	// Replace inline tickboxes
	content = []byte(strings.Replace(string(content), "{box} (if box ticked)", TICKBOX, ANY))

	// Add sheet at the end
	if *addSheet {
		fmt.Print("Adding character sheet... ")
		file, errOpen := os.Open(filepath.Join(dir, SHEET_PDF_NAME))
		check(errOpen)
		pdf, errDocument := fitz.NewFromReader(file)
		check(errDocument)
		sheet1, errConvert := pdf.Image(0)
		check(errConvert)
		sheet2, errConvert := pdf.Image(1)
		check(errConvert)
		var errSave error
		errSave = save(sheet1, filepath.Join(dir, SHEET1_NAME))
		check(errSave)
		errSave = save(sheet2, filepath.Join(dir, SHEET2_NAME))
		check(errSave)
		content = slices.Concat(content, []byte(fmt.Sprintf(FORMAT_ATTACHMENT, filepath.Join(dir, SHEET1_NAME), stripExt(SHEET1_NAME))))
		content = slices.Concat(content, []byte(fmt.Sprintf(FORMAT_ATTACHMENT, filepath.Join(dir, SHEET2_NAME), stripExt(SHEET2_NAME))))
		fmt.Println("done")
	} else {
		fmt.Println("No character sheet was added as per requested.")
	}

	// Write all to output file
	_, errWrite := output.Write(content)
	check(errWrite)
	fmt.Printf("\nDone! Results written to %s\n", *outputName)
}

func save(img *image.RGBA, filename string) error {
	file, errCreate := os.Create(filename)
	if errCreate != nil {
		return errCreate
	}
	defer file.Close()
	var o jpeg.Options
	errEncode := jpeg.Encode(file, img, &o)
	if errEncode != nil {
		return errEncode
	}
	return nil
}

func stripExt(s string) string {
	return strings.TrimSuffix(s, filepath.Ext(s))
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(generateCode(err))
	}
}

func generateCode(err error) (code int) {
	for i := 0; i < 10 && i < len(fmt.Sprint(err)); i++ {
		code += int(fmt.Sprint(err)[i])
	}
	return
}

const FIRST_SECTION = "New.xml"
const A = -1
const B = 1
func betterSort(a, b string) int {
	// Check if one of the paragraph is the first (there can only be one first paragraph since it's based on filename)
	switch {
		case a == FIRST_SECTION:
			return A
		case b == FIRST_SECTION:
			return B
	}

	// Pure numbers go last
	n1, e1 := strconv.Atoi(strings.TrimSuffix(a, filepath.Ext(a)))
	n2, e2 := strconv.Atoi(strings.TrimSuffix(b, filepath.Ext(b)))
	switch {
		case (e1 == nil && e2 == nil):
			return n1 - n2
		case (e1 == nil && e2 != nil):
			return B
		case (e1 != nil && e2 == nil):
			return A
	}

	// Images go after all other non-numbered sections
	switch {
		case filepath.Ext(a) == IMAGE_EXT && filepath.Ext(b) != IMAGE_EXT:
			return B
		case filepath.Ext(b) == IMAGE_EXT && filepath.Ext(a) != IMAGE_EXT:
			return A
		default:
			return strings.Compare(a, b)
	}
}

func capitalize(in string) (out string) {
	for _, w := range strings.Fields(in) {
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		out += string(runes) + " "
	}
	out = out[:len(out)-1]
	return
}

const ADVENTURERS = "Adventurers.xml"
const ADVENTURER_FORMAT =
`
`

const (
	READING_CONTENT = 0
	EXPECTING_ELEMENT = 1
	OPENING_ELEMENT = 2
	CLOSING_ELEMENT = 3
	EXPECTING_ATTRIBUTE = 4
	READING_ATTRIBUTE = 5
	EXPECTING_VALUE = 6
	READING_VALUE = 7
	SKIPPING_ELEMENT = 8
)
const (
	EOL = "\n"
	HISTORY_LENGTH = 50
	POSITION =
`[ Byte %d | Line %d | Mode %d | ]
Buffer contents: [name: %s, attr: %s, value: %s]
Stack contents: %s
Last few characters: %s`
)
func parse(filename string) (output string, err error) {
	var stack stack
	var mode byte
	var name, attr, value string
	var byteCount, lineCount int
	var history []rune
	var depth int
	file, errOpen := os.Open(filename)
	check(errOpen)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanRunes)
	for scanner.Scan() {
		// Get byte
		c := scanner.Text()
		// Update position (for debugging)
		byteCount++
		if c == EOL {
			lineCount++
		}
		// Update history
		history = append(history, []rune(c)[0])
		if len(history) > HISTORY_LENGTH {
			history = history[1:]
		}
		position := fmt.Sprintf(POSITION, byteCount, lineCount, mode, name, attr, value, stack, string(history))

		// Operate this iteration
		switch mode {
			case EXPECTING_ELEMENT:
				switch c {
					case " ":
						continue
					case "?", "!":
						depth = 1
						mode = SKIPPING_ELEMENT
					case ">":
						err = errors.New(fmt.Sprintf("Element was opened and closed immediately %s", position))
						return
					case "/":
						name = ""
						mode = CLOSING_ELEMENT
					default:
						mode = OPENING_ELEMENT
						name = c
				}
			case OPENING_ELEMENT:
				switch c {
					case " ":
						stack.addElement(name)
						mode = EXPECTING_ATTRIBUTE
					case ">":
						stack.addElement(name)
						mode = READING_CONTENT
					case "/":
						stack.addElement(name)
						mode = CLOSING_ELEMENT
					default:
						name += c
				}
			case CLOSING_ELEMENT:
				switch c {
					case " ":
						continue
					case ">":
						if name != stack.Name() {
							err = errors.New(fmt.Sprintf("Tried to close an element that couldn't be closed %s", position))
							return
						} else {
							stack.popElement(&output)
							name = ""
							mode = READING_CONTENT
						}
					default:
						name += c
				}
			case EXPECTING_ATTRIBUTE:
				switch c {
					case ">":
						mode = READING_CONTENT
					case " ":
						continue
					case "/":
						mode = CLOSING_ELEMENT
					default:
						attr = c
						mode = READING_ATTRIBUTE
				}
			case READING_ATTRIBUTE:
				switch c {
					case "=":
						mode = EXPECTING_VALUE
					default:
						attr += c
				}
			case EXPECTING_VALUE:
				switch c {
					case "\"":
						value = ""
						mode = READING_VALUE
				}
			case READING_VALUE:
				switch c {
					case "\"":
						stack.addAttribute(attr, value)
						mode = EXPECTING_ATTRIBUTE
					default:
						value += c
				}
			case READING_CONTENT:
				switch c {
					case "<":
						mode = EXPECTING_ELEMENT
					default:
						stack.extendContent(c, &output)
				}
			case SKIPPING_ELEMENT:
				switch c {
					case "<":
						depth++
					case ">":
						depth--
						if depth == 0 {
							mode = READING_CONTENT
						}
				}
		}
	}
	return
}

func (s stack)String() (output string) {
	if len(s) > 0 {
		output += "["
		for _, e := range s {
			output += e.Name
			output += ", "
		}
		output = output[:len(output)-2]
		output += "]"
	} else {
		return "[]"
	}
	return
}

func (s *stack)addElement(name string) {
	var e element
	e.Name = name
	e.Attributes = make(map[string]string)
	*s = append(*s, e)
}

func (s *stack)addAttribute(key, value string) {
	(*s)[len(*s)-1].Attributes[key] = value
}

const SECTION = "section"
func (s *stack)popElement(output *string) {
	processedElement := replace((*s)[len(*s)-1])
	name := (*s)[len(*s)-1].Name
	*s = (*s)[0:len(*s)-1]
	switch {
		case name == SECTION:
			*output += processedElement
		case len(*s) > 0:
			(*s)[len(*s)-1].Content += processedElement
	}
}

func (s *stack)extendContent(c string, output *string) {
	if len(*s) > 0 {
		(*s)[len(*s)-1].Content += c
	} else {
		*output += c
	}
}

func (s *stack)Name() string {
	if len(*s) > 0 {
		return (*s)[len(*s)-1].Name
	} else {
		return ""
	}
}

func (e element) String() (output string) {
	output += "\n<"
	output += e.Name
	for attribute, value := range e.Attributes {
		output += " "
		output += attribute
		output += "=\""
		output += value
		output += "\""
	}
	output += ">\n\t"
	output += e.Content
	output += "\n</"
	output += e.Name
	output += ">"
	return
}

// THE GREAT REPLACING GALORE

const GOTO_FORMAT =	// in.Attributes["section"], content
`<a href="#%s">%s</a>`
const GOTO_AUTOFILL =
`turn to %s`		// in.Attributes["section"]

const SECTION_FORMAT =	// id, in.Attributes["name"], tickboxes, in.Content	// Classes: page
`
<div class="page">
	<h2 id="%s"><span class="section-title">%s</span><span class="tickboxes">%s</span></h2>
	%s
</div>
`
const TICKBOX = "◻"

const CHOICES_FORMAT =	// in.Content	// Classes: choices
`<table class="choices">
%s
</table>`

const CHOICE_FORMAT =	// in.Content, in.Attributes["section"], in.Attributes["section"]	// Classes: choice-text, choice-selection
`	<tr>
		<td class="choice-text">%s</td>
		<td class="choice-section"><a href="#%s">► turn to %s</a></td>
	</tr>`

const DIFFICULTY_FORMAT =	// in.Attributes["ability"], in.Attributes["difficulty"]	// Classes: ability, difficulty
`Make an ability check on your <span class="ability">%s</span> with difficulty <span class="difficulty">%s</span>`

const RANDOM_FORMAT =	// dice	// Classes: random
`<span class="random">Roll %s dice.</span>`

const RANKCHECK_FORMAT =	// dice	// Classes: rankcheck
`<span class="rankcheck">Roll %s dice and try to score lower than your Rank</span>`

const OUTCOMES_FORMAT =
`<table class="outcomes">
%s
</table>`

const OUTCOME_FORMAT =
`	<tr>
		<td class="outcome-text">%s</td>
		<td class="outcome-section">%s</td>
	</tr>`

const EQUIPMENT_FORMAT =	// in.Attributes["name"]	// Classes: weapon
`<span class="weapon">%s</span>`

const LOSE_FORMAT =	// content	// Classes: lose
`<span class="lose">%s</span>`
const LOSE_AUTOFILL =	// amount, name
`lose %s %s`

const GAIN_FORMAT =	// content	// Classes: gain
`<span class="gain">%s</span>`
const GAIN_AUTOFILL =	// amount, name
`gain %s %s`

const P_FORMAT =	// in.Name, in.Content
`<p class="%s">
	%s
</p>`

const SPAN_FORMAT =	// in.Name, in.Content
` <span class="%s">%s</span>`

type Group struct {
	Text string `xml:",innerxml"`
}

const FIGHT_FORMAT =	// , in.Attributes["name], in.Attributes["combat"], in.Attributes["defence"], in.Attributes["stamina"]	// Classes: a lot
`<table class="fight">
	<tr class="fight-header">
		<th colspan="3" class="fight-name">%s</th>
	</tr>
	<tr class="fight-stats">
		<td class="fight-combat">Combat: %s</td>
		<td class="fight-defence">Defence: %s</td>
		<td class="fight-stamina">Stamina: %s</td>
	</tr>
</table>`

const RESURRECTION_FORMAT =	// Classes: resurrection
`<span class="resurrection">%s</span>`
const RESURRECTION_AUTOFILL =	// in.Attributes["god"], in.Attributes["book"], in.Attributes["section"], in.Attributes["text"]
`Resurrection of %s: Book %s, Section %s (%s)`

const MARKET_FORMAT =
`<table class="market">
	<tr class="market-header-top">
		<th colspan="4">Item</th>
		<th colspan="1">Buy Price</th>
		<th colspan="1">Sell Price</th>
	</tr>
%s
</table>`
const MARKET_HEADER_FORMAT =
`<tr>
	<th colspan="6">%s</th>
</tr>`
const EQUIPMENT_MAKET_FORMAT =
`<tr class="market-item">
	<td class="market-item-name" colspan="4">%s</td>
	<td class="market-item-buy" colspan="1">%s</td>
	<td class="market-item-sell" colspan="1">%s</td>
</tr>`

func replace(in element) (out string) {
	if in.Attributes["hidden"] == "t" {
		out = ""
		return
	}
	switch in.Name {
		case "goto":
			var content string
			if in.Content == "" {
				content = fmt.Sprintf(GOTO_AUTOFILL, in.Attributes["section"])
			} else {
				content = in.Content
			}
			if _, ok := in.Attributes["book"]; ok {
				out = fmt.Sprintf(GOTO_FORMAT, "", content)
			} else {
				out = fmt.Sprintf(GOTO_FORMAT, in.Attributes["section"], content)
			}
		case "section":
			var tickboxes string
			var id string
			var boxCount int
			var ok bool
			if _, ok = in.Attributes["boxes"]; ok {
				boxCount, _ = strconv.Atoi(in.Attributes["boxes"])
				for i := 0; i < boxCount; i++ {
					tickboxes += " " + TICKBOX
				}
			}
			if profession, ok := in.Attributes["profession"]; ok {
				in.Content = (printStats(profession) + in.Content)
				id = strings.Fields(in.Attributes["name"])[0]
			} else {
				id = in.Attributes["name"]
			}
			out = fmt.Sprintf(SECTION_FORMAT, id, in.Attributes["name"], tickboxes, in.Content)
		case "choices":
			out = fmt.Sprintf(CHOICES_FORMAT, in.Content)
		case "choice":
			out = fmt.Sprintf(CHOICE_FORMAT, in.Content, in.Attributes["section"], in.Attributes["section"])
		case "outcomes":
			out = fmt.Sprintf(OUTCOMES_FORMAT, in.Content)
		case "outcome":
			var content string
			if in.Content == "" {
				content = fmt.Sprintf(GOTO_FORMAT, in.Attributes["section"], in.Attributes["section"])
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(OUTCOME_FORMAT, in.Attributes["range"], content)
		case "success", "failure":
			var content string
			if in.Content == "" {
				content = fmt.Sprintf(GOTO_FORMAT, in.Attributes["section"], in.Attributes["section"])
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(OUTCOME_FORMAT, capitalize(in.Name), content)
		case "difficulty":
			out = fmt.Sprintf(DIFFICULTY_FORMAT, in.Attributes["ability"], in.Attributes["level"])
		case "rankcheck":
			out = fmt.Sprintf(RANKCHECK_FORMAT, in.Attributes["dice"])
		case "random":
			var dice string
			var ok bool
			if dice, ok = in.Attributes["dice"]; !ok {
				dice = "2"
			}
			if in.Content == "" {
				out = fmt.Sprintf(RANDOM_FORMAT, dice)
			} else {
				out = in.Content
			}
		case "weapon", "armor", "item", "tool":
			if in.Content == "" {
				var name string
				name = in.Attributes["name"]
				if name == "" {
					name = capitalize(in.Name)
				}
				if bonus, ok := in.Attributes["bonus"]; ok {
					if ability, ok := in.Attributes["ability"]; ok {
						name += fmt.Sprintf(" (+%s to %s rolls)", bonus, ability)
					} else {
						name += fmt.Sprintf(" (+%s)", bonus)
					}
				}
				if sell, ok := in.Attributes["sell"]; ok {
					buy := in.Attributes["buy"]
					if buy == "" {
						buy = "-"
					}
					out = fmt.Sprintf(EQUIPMENT_MAKET_FORMAT, name, buy, sell)
				} else {
					out = fmt.Sprintf(EQUIPMENT_FORMAT, name)
				}
			} else {
				out = in.Content
			}
		case "lose":
			var content string
			if in.Content == "" {
				var name, amount string
				for name, amount = range in.Attributes {
					break
				}
				content = fmt.Sprintf(LOSE_AUTOFILL, amount, name)
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(LOSE_FORMAT, content)
		case "gain":
			var content string
			if in.Content == "" {
				var name, amount string
				for name, amount = range in.Attributes {
					break
				}
				content = fmt.Sprintf(GAIN_AUTOFILL, amount, name)
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(GAIN_FORMAT, content)
		case "buy", "sell":
			var content string
			var ok bool
			if in.Content == "" {
				if content, ok = in.Attributes["item"]; !ok {
					content = in.Attributes["shards"] + " shards"
				}
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(SPAN_FORMAT, in.Name, content)
		case "market":
			out = fmt.Sprintf(MARKET_FORMAT, in.Content)
		case "header":
			out = fmt.Sprintf(MARKET_HEADER_FORMAT, in.Attributes["type"])
		case "trade":
			var name, buy, sell string
			var ok bool
			if name, ok = in.Attributes["ship"]; ok {
				if crew, ok := in.Attributes["initialCrew"]; ok {
					name += " (initial crew: " + crew + ")"
				}
			} else if name, ok = in.Attributes["cargo"]; !ok {
				for _, name = range in.Attributes {
					break
				}
			}
			if sell, ok = in.Attributes["sell"]; !ok {
				sell = "-"
			}
			if buy, ok = in.Attributes["buy"]; !ok {
				buy = "-"
			}
			out = fmt.Sprintf(EQUIPMENT_MAKET_FORMAT, name, buy, sell)
		case "group":
			var group Group
			xml.Unmarshal([]byte(in.Content), &group)
			out = group.Text
		case "fight":
			out = fmt.Sprintf(FIGHT_FORMAT, in.Attributes["name"], in.Attributes["combat"], in.Attributes["defence"], in.Attributes["stamina"])
		case "resurrection":
			var content string
			if in.Content == "" {
				content = fmt.Sprintf(RESURRECTION_AUTOFILL, in.Attributes["god"], in.Attributes["book"], in.Attributes["section"], in.Attributes["text"])
			} else {
				content = in.Content
			}
			out = fmt.Sprintf(RESURRECTION_FORMAT, content)
		case "p", "text":
			out = in.String()
		case "if":
			out = fmt.Sprintf(P_FORMAT, in.Name, in.Content)
		case "adjust":
			out = ""
		case "tick":
			if in.Content == "" {
				out = fmt.Sprintf(SPAN_FORMAT, in.Name, "tick a box")
			} else {
				out = fmt.Sprintf(SPAN_FORMAT, in.Name, in.Content)
			}
		default:
			if *verbose {
				fmt.Println("Unexpected XML tag:", in.Name)
			}

			out = fmt.Sprintf(SPAN_FORMAT, in.Name, in.Content)
	}
	return
}

// --- ADVENTURERS MANAGEMENT ---

type AdventurersRaw struct {
	XMLName xml.Name `xml:"adventurers"`
	Stamina ParameterRaw `xml:"stamina"`
	Rank ParameterRaw `xml:"rank"`
	Gold ParameterRaw `xml:"gold"`
	Abilities []AbilityRaw `xml:"abilities>profession"`
	Equipment EquipmentRaw `xml:"items"`
	Professions []ProfessionRaw `xml:"starting>adventurer"`
}

type EquipmentRaw struct {
	Items []ItemRaw `xml:",any"`
}

type ParameterRaw struct {
	Value string `xml:"amount,attr"`
}

type AbilityRaw struct {
	Profession string `xml:"name,attr"`
	Content string `xml:",innerxml"`
}

type ItemRaw struct {
	XMLName xml.Name
	Profession string `xml:"profession,attr"`
	Name string `xml:"name,attr"`
	Bonus string `xml:"bonus,attr"`
}

type ProfessionRaw struct {
	PersonName string `xml:"name,attr"`
	Profession string `xml:"profession,attr"`
	Description string `xml:",innerxml"`
}

type Profession struct {
	Name string
	PersonName string
	Description string
	Rank string
	Stamina string
	Gold string
	Abilities []string
	Equipment []Item
}

type Item struct {
	Name string
	Type string
	Bonus string
}

var Starting map[string]Profession

func updateStats(fn string) () {
	// Get data from file
	var data []byte
	file, _ := os.Open(fn)
	defer file.Close()
	data, _ = io.ReadAll(file)

	// Process data
	var startingRaw AdventurersRaw
	errUnmarshal := xml.Unmarshal(data, &startingRaw)
	check(errUnmarshal)

	// Insert data into slice
	Starting = make(map[string]Profession)
	for _, p := range startingRaw.Professions {
		var profession Profession
		profession.Name = p.Profession
		profession.PersonName = p.PersonName
		profession.Description = p.Description
		profession.Rank = startingRaw.Rank.Value
		profession.Stamina = startingRaw.Stamina.Value
		profession.Gold = startingRaw.Gold.Value
		for _, a := range startingRaw.Abilities {
			if a.Profession == profession.Name {
				profession.Abilities = strings.Fields(a.Content)
				break
			}
		}
		for _, e := range startingRaw.Equipment.Items {
			if e.Profession == "" || e.Profession == profession.Name {
				var item Item
				item.Name = e.Name
				item.Bonus = e.Bonus
				item.Type = e.XMLName.Local
				profession.Equipment = append(profession.Equipment, item)
			}
		}
		// Insert the freshly baked profession into the slice
		Starting[profession.Name] = profession
	}
	return
}

/* CLASSES:
 * stats-sheet
 * 	stats-abilities-header
 * 		stats-ability-label
 * 	stats-abilities-values
 * 		stats-ability-value
 * 	stats-common-header
 * 		stats-stamina-label
 * 		stats-rank-label
 * 		stats-gold-label
 * 	stats-common-values
 * 		stats-stamina-value
 * 		stats-rank-value
 * 		stats-gold-value
 * 	equipment-header
 * 		equipment-label
 * 	equipment-value
 * 		equipment-item-type
 * 		equipment-item-name
 */
const STATS_FORMAT =	// p.Name, p.Abilities..., p.Stamina, p.Rank, p.Gold, startingEquip
`<h3 class="profession">
%s
</h3>
<table class="stats-sheet">
<tr class="stats-abilities-header">
<th class="stats-ability-label">Charisma</th>
<th class="stats-ability-label">Combat</th>
<th class="stats-ability-label">Magic</th>
<th class="stats-ability-label">Sanctity</th>
<th class="stats-ability-label">Scouting</th>
<th class="stats-ability-label">Thievery</th>
</tr>
<tr class="stats-abilities-values">
<td class="stats-ability-value">%s</td>
<td class="stats-ability-value">%s</td>
<td class="stats-ability-value">%s</td>
<td class="stats-ability-value">%s</td>
<td class="stats-ability-value">%s</td>
<td class="stats-ability-value">%s</td>
</tr>
<tr class="stats-common-header">
<th class="stats-stamina-label" colspan="2">Stamina</th>
<th class="stats-rank-label" colspan="2">Rank</th>
<th class="stats-gold-label" colspan="2">Gold</th>
</tr>
<tr class="stats-common-values">
<td class="stats-stamina-value" colspan="2">%s</td>
<td class="stats-rank-value" colspan="2">%s</td>
<td class="stats-gold-value" colspan="2">%s</td>
</tr>
<tr class="equipment-header">
<th class="equipment-label" colspan="6">Starting equipment</th>
</tr>
%s
</table>`
const STARTING_EQUIP_FORMAT =
`<tr class="equipment-value">
<th class="equipment-item-type" colspan="2">%s</td>
<td class="equipment-item-name" colspan="4">%s</td>
</tr>`
func printStats(name string) string {
	var p Profession
	var startingEquip string
	var ok bool
	p, ok = Starting[name]
	if !ok {
		fmt.Printf("Found no match for %s!\nThese are the starting professions registered from Adventurers.xml:\n", name)
		for k := range Starting {
			fmt.Println(k)
		}
		fmt.Println("Aborting.")
		os.Exit(666)
	}
	cha, com, mag, san, sco, thi := p.Abilities[0], p.Abilities[1], p.Abilities[2], p.Abilities[3], p.Abilities[4], p.Abilities[5]
	for _, e := range p.Equipment {
		var nameFull string
		if e.Bonus != "" {
			nameFull = e.Name + " (+" + e.Bonus + ")"
		} else {
			nameFull = e.Name
		}
		startingEquip += fmt.Sprintf(STARTING_EQUIP_FORMAT, e.Type, nameFull)
	}
	return fmt.Sprintf(STATS_FORMAT, p.Name, cha, com, mag, san, sco, thi, p.Stamina, p.Rank, p.Gold, startingEquip)
}
