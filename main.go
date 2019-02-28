package main

import (
	_ "database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
)

type eventDocument struct {
	at    time.Time
	name  string
	value string
}

var eventCollection []eventDocument

func main() {
	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:hakaru-pass@tcp(127.0.0.1:13306)/hakaru-db"
	}

	go func() {
		tmp := make([]eventDocument, len(eventCollection))
		copy(tmp, eventCollection)
		eventCollection = []eventDocument{}
		t := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-t.C:
				conn, _ := dbr.Open("mysql", dataSourceName, nil)
				conn.SetMaxOpenConns(5)
				sess := conn.NewSession(nil)
				//tx, err := sess.Begin()
				//if err != nil {
				//	panic(err.Error())
				//}
				//defer tx.RollbackUnlessCommitted()
				stmt := sess.InsertInto("eventlog").
					Columns("at", "name", "value")

				for _, value := range tmp {
					stmt.Record(value)
				}

				buf := dbr.NewBuffer()
				stmt.Build(dialect.MySQL, buf)

				result, err := stmt.Exec()
				if err != nil {
					fmt.Println(err)
				} else {
					count, _ := result.RowsAffected()
					fmt.Println(count)
				}
			}
		}
		t.Stop()
		//tx.Commit()
	}()

	hakaruHandler := func(w http.ResponseWriter, r *http.Request) {
		//db, err := sql.Open("mysql", dataSourceName)
		//if err != nil {
		//	panic(err.Error())
		//}
		//defer db.Close()

		//stmt, e := db.Prepare("INSERT INTO eventlog(at, name, value) values(NOW(), ?, ?)")
		//if e != nil {
		//	panic(e.Error())
		//}
		//
		//defer stmt.Close()

		name := r.URL.Query().Get("name")
		value := r.URL.Query().Get("value")
		eventCollection = append(eventCollection, eventDocument{time.Now(), name, value})

		//_, _ = stmt.Exec(name, value)

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
	}

	http.HandleFunc("/hakaru", hakaruHandler)
	http.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// start server
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
