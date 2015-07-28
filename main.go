package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nsf/termbox-go"

	_ "github.com/lib/pq"
	"github.com/pavlik/fias_xml2postgresql/structures/actual_status"
	"github.com/pavlik/fias_xml2postgresql/structures/address_object"
	"github.com/pavlik/fias_xml2postgresql/structures/address_object_type"
	"github.com/pavlik/fias_xml2postgresql/structures/center_status"
	"github.com/pavlik/fias_xml2postgresql/structures/current_status"
	"github.com/pavlik/fias_xml2postgresql/structures/estate_status"
	"github.com/pavlik/fias_xml2postgresql/structures/house"
	"github.com/pavlik/fias_xml2postgresql/structures/house_interval"
	"github.com/pavlik/fias_xml2postgresql/structures/house_state_status"
	"github.com/pavlik/fias_xml2postgresql/structures/interval_status"
	"github.com/pavlik/fias_xml2postgresql/structures/landmark"
	"github.com/pavlik/fias_xml2postgresql/structures/normative_document"
	"github.com/pavlik/fias_xml2postgresql/structures/normative_document_type"
	"github.com/pavlik/fias_xml2postgresql/structures/operation_status"
	"github.com/pavlik/fias_xml2postgresql/structures/structure_status"
)

func print_tb(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}

func printf_tb(x, y int, fg, bg termbox.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	print_tb(x, y, fg, bg, s)
}

const timeLayout = "2006-01-02 в 15:04"

func progressPrint(msgs [15]string, startTime time.Time, finished bool) {
	color := [15]termbox.Attribute{termbox.ColorWhite,
		termbox.ColorWhite,
		termbox.ColorWhite,
		termbox.ColorWhite,
		termbox.ColorWhite,
		termbox.ColorBlue,
		termbox.ColorBlue,
		termbox.ColorBlue,
		termbox.ColorBlue,
		termbox.ColorBlue,
		termbox.ColorRed,
		termbox.ColorRed,
		termbox.ColorRed,
		termbox.ColorRed,
		termbox.ColorRed}

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	printf_tb(0, 0, termbox.ColorGreen|termbox.AttrBold, termbox.ColorBlack, "Экспорт базы ФИАС в БД PostgreSQL")
	printf_tb(0, 1, termbox.ColorYellow, termbox.ColorBlack, fmt.Sprintf("Количество используемых ядер: %d", runtime.NumCPU()))
	printf_tb(0, 20, termbox.ColorCyan, termbox.ColorBlack, fmt.Sprintf("Конвертация началась %s", startTime.Format(timeLayout)))

	duration := time.Since(startTime)
	if !finished {
		printf_tb(0, 21, termbox.ColorCyan, termbox.ColorBlack, fmt.Sprintf("и уже длится %.0f минут", duration.Minutes()))
	} else {
		printf_tb(0, 21, termbox.ColorGreen, termbox.ColorBlack, fmt.Sprintf("База экспортировалась за %.0f минут. Экспорт завершен.", duration.Minutes()))
	}
	printf_tb(0, 22, termbox.ColorMagenta|termbox.AttrUnderline, termbox.ColorBlack, "Для прерывания экспорта и выхода из программы нажмите CTRL+Q")

	y := 0
	for _, v := range msgs {
		printf_tb(0, y+3, color[y], termbox.ColorDefault, v)
		y++
	}

	termbox.Flush()
}

func initDb() *sqlx.DB {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sqlx.Open("postgres", "user=dev dbname=fias password=dev sslmode=disable")
	checkErr(err, "sqlx.Open failed")

	return db
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var format = flag.String("format", "xml", "File format for import (xml or dbf)")
	flag.Parse()

	// initialize the DbMap
	db := initDb()
	defer db.Close()

	var w sync.WaitGroup

	as_stat := make(chan string, 1000)
	ao_stat := make(chan string, 1000)
	cs_stat := make(chan string, 1000)
	cur_stat := make(chan string, 1000)
	est_stat := make(chan string, 1000)
	house_stat := make(chan string, 1000)
	house_int_stat := make(chan string, 1000)
	house_st_stat := make(chan string, 1000)
	intv_stat := make(chan string, 1000)
	landmark_stat := make(chan string, 1000)
	ndtype_stat := make(chan string, 1000)
	nd_stat := make(chan string, 1000)
	oper_stat := make(chan string, 1000)
	socrbase_stat := make(chan string, 1000)
	str_stat := make(chan string, 1000)

	if *format == "xml" {
		fmt.Println("обработка XML-файлов")
		go actual_status.Export(&w, as_stat, db, format)
		go address_object.Export(&w, ao_stat, db, format)
		go center_status.Export(&w, cs_stat, db, format)
		go current_status.Export(&w, cur_stat, db, format)
		go estate_status.Export(&w, est_stat, db, format)
		go house.Export(&w, house_stat, db, format)
		go house_interval.Export(&w, house_int_stat, db, format)
		go house_state_status.Export(&w, house_st_stat, db, format)
		go interval_status.Export(&w, intv_stat, db, format)
		go landmark.Export(&w, landmark_stat, db, format)
		go normative_document_type.Export(&w, ndtype_stat, db, format)
		go normative_document.Export(&w, nd_stat, db, format)
		go operation_status.Export(&w, oper_stat, db, format)
		go address_object_type.Export(&w, socrbase_stat, db, format)
		go structure_status.Export(&w, str_stat, db, format)

	} else if *format == "dbf" {
		// todo: обработка DBF-файлов
		fmt.Println("обработка DBF-файлов")
	}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)

	var msgs [15]string
	timer := time.After(time.Second * 1)
	go func() {

		startTime := time.Now()

		for {
			select {
			case <-timer:
				progressPrint(msgs, startTime, false)
				timer = time.After(time.Second * 1)
			default:
			}
			select {
			case msgs[0] = <-as_stat:
			case msgs[1] = <-ao_stat:
			case msgs[2] = <-cs_stat:
			case msgs[3] = <-cur_stat:
			case msgs[4] = <-est_stat:
			case msgs[5] = <-house_stat:
			case msgs[6] = <-house_int_stat:
			case msgs[7] = <-house_st_stat:
			case msgs[8] = <-intv_stat:
			case msgs[9] = <-landmark_stat:
			case msgs[10] = <-ndtype_stat:
			case msgs[11] = <-nd_stat:
			case msgs[12] = <-oper_stat:
			case msgs[13] = <-socrbase_stat:
			case msgs[14] = <-str_stat:
			}
		}

		w.Wait()
		progressPrint(msgs, startTime, true)
	}()

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyCtrlQ {
				break loop
			}
			termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
			termbox.Flush()
			//progressPrint(msg1, msg2, msg3, msg4, msg5)
		case termbox.EventError:
			panic(ev.Err)
		}
	}
}
