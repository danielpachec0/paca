package alarm

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
)

type Day struct {
	NameAbbreviation string
	Name             string
	Status           bool
}

type SqlAlarm struct {
	Id      int
	Hours   int
	Minutes int
	Days    [7]Day
	Alarm   bool
	Ac      bool
	Curtain bool
	Owner   int
}

func FetchUserAlarms(id int, db *sql.DB) ([]SqlAlarm, error) {
	var queryString = `SELECT
			a.id, hours, minutes, monday,
			tuesday, wednesday, thursday,
			friday, saturday,
        	sunday, alarm, ac, curtain
		FROM alarms a JOIN days on days.id = a.days WHERE a.user = ?`

	rows, err := db.Query(queryString, id)

	if err != nil {
		return nil, err
	}

	alarms := make([]SqlAlarm, 0)
	for rows.Next() {
		var i SqlAlarm
		i.Days = [7]Day{{"M", "monday", false},
			{"T", "tuesday", false}, {"W", "wednesday", false},
			{"T", "thursday", false}, {"F", "friday", false},
			{"S", "saturday", false}, {"S", "sunday", false}}

		if err := rows.Scan(
			&i.Id,
			&i.Hours, &i.Minutes,
			&i.Days[0].Status, &i.Days[1].Status, &i.Days[2].Status,
			&i.Days[3].Status, &i.Days[4].Status, &i.Days[5].Status, &i.Days[6].Status,
			&i.Alarm, &i.Ac, &i.Curtain,
		); err != nil {
			return nil, err
		}
		alarms = append(alarms, i)
	}
	return alarms, nil
}

func FetchAlarm(alarmId string, db *sql.DB) (SqlAlarm, error) {
	var returnedAlarm SqlAlarm
	row := db.QueryRow("SELECT * FROM alarms a JOIN days on days.id = a.days WHERE a.id = ?", alarmId)

	returnedAlarm.Days = [7]Day{{"M", "monday", false},
		{"T", "tuesday", false}, {"W", "wednesday", false},
		{"T", "thursday", false}, {"F", "friday", false},
		{"S", "saturday", false}, {"S", "sunday", false}}
	var daysID int
	var days int
	if err := row.Scan(
		&returnedAlarm.Id, &returnedAlarm.Hours, &returnedAlarm.Minutes,
		&days,
		&returnedAlarm.Owner,
		&returnedAlarm.Alarm, &returnedAlarm.Ac, &returnedAlarm.Curtain,
		&daysID,
		&returnedAlarm.Days[0].Status, &returnedAlarm.Days[1].Status, &returnedAlarm.Days[2].Status,
		&returnedAlarm.Days[3].Status, &returnedAlarm.Days[4].Status, &returnedAlarm.Days[5].Status, &returnedAlarm.Days[6].Status,
	); err != nil {
		log.Fatal(err)
	}
	return returnedAlarm, nil
}

func CreateAlarm(userId string, form map[string]string, db *sql.DB) error {
	stmt, err := db.Prepare("INSERT into days (monday, tuesday, wednesday, thursday, friday, saturday, sunday) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	result, err := stmt.Exec(
		form["monday"] == "on",
		form["tuesday"] == "on",
		form["wednesday"] == "on",
		form["thursday"] == "on",
		form["friday"] == "on",
		form["saturday"] == "on",
		form["sunday"] == "on",
	)
	if err != nil {
		return err
	}

	dayId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	stmt, err = db.Prepare("INSERT into alarms (user, hours, minutes, days, alarm, ac, curtain) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	hours, _ := strconv.Atoi(form["hours"])
	minutes, _ := strconv.Atoi(form["minutes"])
	curtain := form["curtain"] == "on"
	alarmVal := form["alarm"] == "on"
	ac := form["ac"] == "on"
	_, err = stmt.Exec(userId, hours, minutes, dayId, alarmVal, ac, curtain)
	if err != nil {
		return err
	}

	return nil
}

func UpdateAlarm(alarmId string, form map[string]string, db *sql.DB) error {
	rows, err := db.Query("SELECT id, user, days FROM alarms  WHERE id = ?", alarmId)
	if err != nil {
		log.Fatal(err)
	}
	var returnedId int
	var daysId string
	var owner int
	for rows.Next() {
		err := rows.Scan(&returnedId, &owner, &daysId)
		if err != nil {
			log.Fatal(err)
		}
	}

	if strconv.Itoa(returnedId) != alarmId {
		return errors.New("alarm not found")
		//http.NotFound(writer, request)
	}

	//check if user has access to item
	id := 1
	if owner != id {
		//http.Error(writer, "Forbidden", 403)
		return errors.New("forbidden")
	}

	_, err = db.Exec("UPDATE days SET monday = ?, tuesday = ?, wednesday = ?, thursday = ?, friday = ?,saturday = ?, sunday = ? WHERE id = ?",
		form["monday"] == "on",
		form["tuesday"] == "on",
		form["wednesday"] == "on",
		form["thursday"] == "on",
		form["friday"] == "on",
		form["saturday"] == "on",
		form["sunday"] == "on",
		daysId,
	)

	if err != nil {
		return errors.New("forbidden")
	}

	hours, _ := strconv.Atoi(form["hours"])
	minutes, _ := strconv.Atoi(form["minutes"])
	curtain := form["curtain"] == "on"
	alarmVal := form["alarm"] == "on"
	ac := form["ac"] == "on"
	_, err = db.Exec("UPDATE alarms SET hours = ?, minutes= ?, days= ?, alarm= ?, ac= ?, curtain = ? WHERE  id = ?",
		hours, minutes, daysId, alarmVal, ac, curtain, alarmId)
	if err != nil {
		return errors.New("forbidden")
	}
	return nil
}

func DeleteAlarm(alarmId string, db *sql.DB) error {
	rows, err := db.Query("SELECT id, user  FROM alarms  WHERE id = ?", alarmId)
	if err != nil {
		return errors.New("internal")
	}

	var returnedId int
	var owner int

	for rows.Next() {
		err := rows.Scan(&returnedId, &owner)
		if err != nil {
			log.Fatal(err)
		}
	}

	if strconv.Itoa(returnedId) != alarmId {
		return errors.New("not found")
	}

	if owner != 1 {
		return errors.New("forbidden")
	}

	_, err = db.Exec("DELETE FROM alarms WHERE id = ?", alarmId)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
