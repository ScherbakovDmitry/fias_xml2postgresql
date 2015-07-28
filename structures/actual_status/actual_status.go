package actual_status

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pavlik/fias_xml2postgresql/helpers"
)

// Статус актуальности ФИАС
type XmlObject struct {
	XMLName   xml.Name `xml:"ActualStatus"`
	ActStatId int      `xml:"ACTSTATID,attr"`
	Name      string   `xml:"NAME,attr"`
}

// схема таблицы в БД

const tableName = "actstat"
const elementName = "ActualStatus"

const schema = `CREATE TABLE ` + tableName + ` (
    act_stat_id INT UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
		PRIMARY KEY (act_stat_id));`

func (item XmlObject) String() string {
	return fmt.Sprintf("\t ActStatId : %d - Name : %s \n", item.ActStatId, item.Name)
}

func Export(w *sync.WaitGroup, c chan string, db *sqlx.DB, format *string) {
	w.Add(1)
	defer w.Done()

	helpers.DropAndCreateTable(schema, tableName, db)

	var format2 string
	format2 = *format
	fileName, err2 := helpers.SearchFile(tableName, format2)
	if err2 != nil {
		fmt.Println("Error searching file:", err2)
		return
	}

	pathToFile := format2 + "/" + fileName

	// Подсчитываем, сколько элементов нужно обработать
	//_, err := helpers.CountElementsInXML(pathToFile, elementName)
	// if err != nil {
	// 	fmt.Println("Error counting elements in XML file:", err)
	// 	return
	// }
	// fmt.Println("\nВ ", elementName, " содержится ", countedElements, " строк")

	xmlFile, err := os.Open(pathToFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)
	total := 0
	var inElement string
	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			inElement = se.Name.Local

			if inElement == elementName {
				total++
				var item XmlObject
				// decode a whole chunk of following XML into the
				// variable item which is a ActualStatus (se above)
				err = decoder.DecodeElement(&item, &se)
				if err != nil {
					fmt.Println("Error in decode element:", err)
					return
				}
				query := "INSERT INTO " + tableName + " (act_stat_id, name) VALUES ($1, $2)"
				db.MustExec(query, item.ActStatId, item.Name)

				s := strconv.Itoa(total)

				c <- elementName + " " + s + " rows affected"
				//fmt.Printf(item.String())
			}
		default:
		}

	}

	//fmt.Printf("\nВсего в "+elementName+" обработано %d строк\n", total)
}
