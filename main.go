package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type PageData struct {
    PageTitle string
    Session   string
}

type Response struct {
    Message string `json:"message"`
    Code    int    `json:"code"`
}

var db *sql.DB

func getDB() *sql.DB {
    if db != nil {
        db.Close()
    }

    var err error
    db, err = sql.Open("sqlite3", "file:db.sqlite")
    if err != nil {
        panic(err)
    }

    return db
}

func CountLogs(query string, args ...any) int {
    var length int

    rows, _ := getDB().Query(query, args...)

    defer rows.Close()
    if rows.Next() {
        rows.Scan(&length)
    }

    return length
}

func Exec(query string, args ...any) int64 {
    stmt, err := getDB().Prepare(query)
    if err != nil {
        panic(err)
    }

    res, err := stmt.Exec(args...)
    if err != nil {
        panic(err)
    }
    count, err := res.RowsAffected()
    if err != nil {
        panic(err)
    }
    
    return count
}

func getSessionOrRedirct(w http.ResponseWriter, r *http.Request) (string, error) {
    session := r.URL.Query().Get("s")

    if session != "" {
        fmt.Println("Session loaded: ", session)
        return session, nil
    }
    
    http.Redirect(w, r, "/", http.StatusSeeOther)
    return "", errors.New("Missing session Parameters")
}

func main() {
    

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        session := uuid.New()
        fmt.Println("Generating new Session: ", session.String())
        
        http.Redirect(w, r, "/diner?s=" + session.String(), http.StatusSeeOther)
    })

    http.HandleFunc("/diner", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("Rendering Diner URL")
        tpl := template.Must(template.ParseFiles("./layout.html"))

        session, err := getSessionOrRedirct(w, r)
        if err != nil {
            return
        }

        length := CountLogs("SELECT COUNT(*) AS length FROM logs WHERE ID=? AND end_at IS NOT NULL;", session)
        if length > 0 {
            fmt.Println("Session already complete generating new one")
            http.Redirect(w, r, "/", http.StatusSeeOther)
            return
        }

        data := PageData{
            PageTitle: "Diner Time",
            Session: session,
        }

        tpl.Execute(w, data)
    })

    http.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("/api/start")
        session, err := getSessionOrRedirct(w, r)
        if err != nil {
            return
        }
        
        
        length := CountLogs("SELECT COUNT(*) AS length FROM logs WHERE ID=?;", session)
        if length > 0 {
            errorResponse := Response {
                Message: fmt.Sprintf("Session %s already started", session),
                Code: 100,
            }
            fmt.Println(errorResponse.Message)
            json.NewEncoder(w).Encode(errorResponse)
            return
        }

        now := time.Now()
        res := Exec(
            "INSERT INTO logs (ID, start_at, date) VALUES (?, ?, ?)", 
            session, 
            now.Format(time.TimeOnly),
            now.Format(time.DateOnly),
        )
        
        success := Response {
            Message: fmt.Sprintf("Success: timer is started"),
            Code: int(res),
        }

        json.NewEncoder(w).Encode(success)
    })
    
    http.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("/api/stop")
        session, err := getSessionOrRedirct(w, r)
        if err != nil {
            return
        }

        length := CountLogs("SELECT COUNT(*) AS length FROM logs WHERE ID=? AND end_at IS NULL;", session)

        if length == 0 {
            errorResponse := Response {
                Message: fmt.Sprintf("Session %s is not started", session),
                Code: 100,
            }
            fmt.Println(errorResponse.Message)
            json.NewEncoder(w).Encode(errorResponse)
            return
        }

        res := Exec(
            "UPDATE logs SET end_at=? WHERE ID=?", 
            time.Now().Format(time.TimeOnly),
            session,
        )

        success := Response {
            Message: fmt.Sprintf("Success: timer is stop"),
            Code: int(res),
        }

        json.NewEncoder(w).Encode(success)
    })
    
    http.HandleFunc("/api/feeling", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("/api/feeking")
        session, err := getSessionOrRedirct(w, r)
        if err != nil {
            return
        }
        
        feeling := r.URL.Query().Get("feeling")
        if feeling == "" {
            err := Response {
                Message: "Error missing feeling params",
                Code: 200,
            }
            json.NewEncoder(w).Encode(err)
        }
            
        length := CountLogs("SELECT COUNT(*) AS length FROM logs WHERE ID=? AND feeling IS NULL;", session)
        
        if length == 0 {
            errorResponse := Response {
                Message: fmt.Sprintf("Session %s feeling already set", session),
                Code: 100,
            }
            fmt.Println(errorResponse.Message)
            json.NewEncoder(w).Encode(errorResponse)
            return
        }

        res := Exec(
            "UPDATE logs SET feeling=? WHERE ID=?", 
            feeling,
            session,
        )

        success := Response {
            Message: fmt.Sprintf("Success: feeling saved"),
            Code: int(res),
        }

        json.NewEncoder(w).Encode(success)
    })
    
    fs := http.FileServer(http.Dir("client/"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    http.ListenAndServe(":1201", nil)
}
