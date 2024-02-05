package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"github.com/danielpachec0/paca/internal/alarm"
	"github.com/danielpachec0/paca/internal/auth"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	port := "8080"

	alarmMux := http.NewServeMux()

	alarmMux.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			tpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				log.Fatal(err)
			}
			err = tpl.Execute(writer, nil)
			if err != nil {
				log.Fatal(err)
			}
		} else if request.Method == http.MethodPost {
			err := request.ParseForm()
			if err != nil {
				log.Fatal(err)
			}

			db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
			if err != nil {
				log.Fatal(err)
			}

			queryString := "SELECT id, password_digest FROM users WHERE mail = '" + request.FormValue("mail") + "'"
			rows, err := db.Query(queryString)
			if err != nil {
				log.Fatal(err)
			}

			var id int
			var passwordDigest string
			for rows.Next() {
				if err := rows.Scan(&id, &passwordDigest); err != nil {
					log.Fatal(err)
				}
			}

			if request.FormValue("password") != passwordDigest {
				return
			}
			expiration := time.Now().Add(24 * time.Hour)
			randomBytes := make([]byte, 8)
			_, err = rand.Read(randomBytes)
			if err != nil {
				log.Fatal(err)
			}
			token := base64.URLEncoding.EncodeToString(randomBytes)
			cookie := http.Cookie{
				Name:     "_paca_token",
				Value:    token,
				Expires:  expiration,
				SameSite: 1,
				HttpOnly: true,
				//Secure:   true, //set to https only
				Path: "/",
			}
			http.SetCookie(writer, &cookie)
			sessions[token] = id
			http.Redirect(writer, request, "/alarm", http.StatusSeeOther)

		} else {
			http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	alarmMux.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("styles"))))
	alarmMux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	alarmMux.HandleFunc("/favicon.ico", faviconHandler)

	alarmMux.HandleFunc("/alarm", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			getAlerts(writer, request) //List all alarms
		} else {
			http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	alarmMux.HandleFunc("/alarm/new", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			newAlertPage(writer, request)
		} else if request.Method == http.MethodPost {
			postAlert(writer, request)
		} else {
			http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	alarmMux.HandleFunc("/alarm/", func(writer http.ResponseWriter, request *http.Request) {
		segments := strings.Split(request.URL.Path, "/")
		if len(segments) < 3 {
			http.NotFound(writer, request)
			return
		}

		idStr := segments[2]

		_, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(writer, "Invalid ID", http.StatusBadRequest)
			return
		}

		switch request.URL.Path {
		case "/alarm/" + idStr:
			if request.Method == http.MethodGet {
				alertDetails(idStr, writer, request)
			} else {
				http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/alarm/" + idStr + "/edit":
			if request.Method == http.MethodGet {
				editAlertDialog(idStr, writer, request)
			} else if request.Method == http.MethodPost {
				editAlert(idStr, writer, request)
			} else {
				http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/alarm/" + idStr + "/delete":
			if request.Method == http.MethodGet {
				alertDeleteDialog(writer, request)
			} else if request.Method == http.MethodPost {
				deleteAlert(idStr, writer, request)
			} else {
				http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.NotFound(writer, request)
		}
	})

	err := http.ListenAndServe(":"+port, alarmMux)
	if err != nil {
		log.Fatal("Server could not be started")
	}
}

var sessions = map[string]int{}

func transformDateComp(input int) string {
	var result string
	result = strconv.Itoa(input)
	if input < 10 {
		result = "0" + result
	}
	return result
}

func getAlerts(writer http.ResponseWriter, request *http.Request) {
	id, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	alarms, err := alarm.FetchUserAlarms(id, db)
	if err != nil {
		log.Fatal(err)
	}

	tpl, err := template.ParseFiles("templates/alarms.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	err = tpl.Execute(writer, alarms)
	if err != nil {
		log.Fatal(err)
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "assets/favicon.ico")
}

func postAlert(writer http.ResponseWriter, request *http.Request) {
	id, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	err = request.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}

	formData := make(map[string]string)
	for key, values := range request.Form {
		if len(values) > 0 {
			formData[key] = values[0]
		}
	}
	err = alarm.CreateAlarm(strconv.Itoa(id), formData, db)
	if err != nil {
		log.Fatal(err)
	}

	//insert dialog
	http.Redirect(writer, request, "/alarm", 303)
}

func newAlertPage(writer http.ResponseWriter, request *http.Request) {
	_, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	funcMap := map[string]interface{}{
		"customFunction": func(input int) string {
			if input < 10 {
				return "0" + strconv.Itoa(input)
			}
			return strconv.Itoa(input)
		},
	}

	tpl, err := template.New("new_alarm.gohtml").
		Funcs(funcMap).
		ParseFiles("templates/new_alarm.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	type dataType struct {
		HoursArr   [24]int
		MinutesArr [60]int
		aux        func(int) string
	}

	data := dataType{
		HoursArr:   [24]int{},
		MinutesArr: [60]int{},
		aux:        transformDateComp,
	}

	err = tpl.Execute(writer, data)
	if err != nil {
		log.Fatal(err)
	}
}

func editAlert(alarmId string, writer http.ResponseWriter, request *http.Request) {
	_, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}

	err = request.ParseForm()

	formData := make(map[string]string)
	for key, values := range request.Form {
		if len(values) > 0 {
			formData[key] = values[0]
		}
	}

	err = alarm.UpdateAlarm(alarmId, formData, db)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	//insert dialog
	http.Redirect(writer, request, "/alarm", 303)
}

func editAlertDialog(alarmId string, writer http.ResponseWriter, request *http.Request) {
	id, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	returnedAlarm, err := alarm.FetchAlarm(alarmId, db)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	//check if user has access to item
	if returnedAlarm.Owner != id {
		http.Error(writer, "Forbidden", 403)
		return
	}

	funcMap := map[string]interface{}{
		"customFunction": func(input int) string {
			if input < 10 {
				return "0" + strconv.Itoa(input)
			}
			return strconv.Itoa(input)
		},
	}

	tpl, err := template.New("editAlarm.gohtml").
		Funcs(funcMap).
		ParseFiles("templates/editAlarm.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	type dataType struct {
		HoursArr   [24]int
		MinutesArr [60]int
		aux        func(int) string
	}

	data := dataType{
		HoursArr:   [24]int{},
		MinutesArr: [60]int{},
		aux:        transformDateComp,
	}

	type test struct {
		Aux   dataType
		Alarm alarm.SqlAlarm
	}
	testObj := test{data, returnedAlarm}

	err = tpl.Execute(writer, testObj)
	if err != nil {
		log.Fatal(err)
	}
}

func alertDeleteDialog(writer http.ResponseWriter, request *http.Request) {
	_, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	tpl, err := template.ParseFiles("templates/deleteAlarmDialog.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	err = tpl.Execute(writer, "1")
}

func deleteAlert(alarmId string, writer http.ResponseWriter, request *http.Request) {
	_, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}

	err = alarm.DeleteAlarm(alarmId, db)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(writer, request, "/alarm", 303)
}

func alertDetails(alarmId string, writer http.ResponseWriter, request *http.Request) {
	id, err := auth.Auth(sessions, request.Cookies())
	if err != nil {
		http.Error(writer, err.Error(), 401)
		return
	}

	db, err := sql.Open("sqlite3", "/home/daniel/Projects/paca/data/database.db")
	if err != nil {
		log.Fatal(err)
	}

	returnedAlarm, err := alarm.FetchAlarm(alarmId, db)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	if returnedAlarm.Owner != id {
		http.Error(writer, "Forbidden", 403)
		return
	}

	tpl, err := template.ParseFiles("templates/alarmDetails.gohtml")
	if err != nil {
		http.Error(writer, err.Error(), 500)
	}

	err = tpl.Execute(writer, returnedAlarm)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}
}
