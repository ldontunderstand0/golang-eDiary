package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	_ "github.com/go-sql-driver/mysql"
)

var cookieStore = sessions.NewCookieStore([]byte("secret"))

const cookieName = "MyCookie"

type sesKey int

const (
	sesKeyLogin sesKey = iota
)

type Shape interface {
	get_labels() []string
}

type User struct {
	Id         int    `json:"id"`
	Group_id   int    `json:"group_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Login      string `json:"login"`
	Password   string `json:"password"`
	Is_teacher bool   `json:"is_teacher"`
	Is_admin   bool   `json:"is_admin"`
}

func (u User) get_labels() []string {
	return []string{"id", "group_id", "name", "email", "login", "password", "is_teacher", "is_admin"}
}

type Group struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Shortname string `json:"shortname"`
}

func (g Group) get_labels() []string {
	return []string{"id", "name", "shortname"}
}

type Subject struct {
	Id        int    `json:"id"`
	User_id   int    `json:"user_id"`
	Name      string `json:"name"`
	Shortname string `json:"shortname"`
}

func (s Subject) get_labels() []string {
	return []string{"id", "user_id", "name", "shortname"}
}

type SubjectGroup struct {
	Id         int `json:"id"`
	Group_id   int `json:"group_id"`
	Subject_id int `json:"subject_id"`
}

func (sg SubjectGroup) get_labels() []string {
	return []string{"id", "group_id", "subject_id"}
}

type Lession struct {
	Id               int    `json:"id"`
	Subject_group_id int    `json:"subject_group_id"`
	Kind             string `json:"kind"`
	Date             string `json:"date"`
}

func (l Lession) get_labels() []string {
	return []string{"id", "subject_group_id", "kind", "date"}
}

type LessionUser struct {
	Id         int `json:"id"`
	Lession_id int `json:"lession_id"`
	User_id    int `json:"user_id"`
	Presence   int `json:"presence"`
	Grade      int `json:"grade"`
}

func (lu LessionUser) get_labels() []string {
	return []string{"id", "lession_id", "user_id", "presence", "grade"}
}

var user User
var group Group
var subject Subject
var subject_group SubjectGroup
var lession Lession
var lession_user LessionUser
var model_map = map[string]Shape{
	"users":           user,
	"class":           group,
	"subjects":        subject,
	"subjects_groups": subject_group,
	"lessions":        lession,
	"lessions_users":  lession_user,
}

type PairString struct {
	String1 string
	String2 string
}

type TableForm struct {
	Username string
	Name     string
	Titles   []string
	Contents [][]string
}

type CrudForm struct {
	Form  string
	Value string
}

type CrudTableForm struct {
	Username string
	Name     string
	Crud     []CrudForm
	Type     string
}

type ListForm struct {
	Username string
	Type     string
	Name     string
	Btn_name string
	List     []PairString
}

type ElemForm struct {
	Id, Is_grade int
	Grade        string
}

type StatsForm struct {
	Subject string
	Avg     float64
	Perc    int
}

type MainStatsForm struct {
	Username  string
	StatsList []StatsForm
}

type RowForm struct {
	Name  string
	Perc  int
	Avg   float64
	Grade []ElemForm
}

type MainTableForm struct {
	Username    string
	SubjectName string
	RowList     []RowForm
	LessionList []Lession
}

type UpdateForm struct {
	Username       string
	GroupName      string
	GroupShortname string
	Subname        string
	Kind           int
}

type UserForm struct {
	Name  string
	Login string
	Role  string
	Auth  bool
}

func index(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}
	t, err := template.ParseFiles("templates/index.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "index", nil)
}

func stats(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role != "student" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.QueryRow("SELECT id, group_id, name FROM users WHERE login = '"+vars["login"]+"';").Scan(&user.Id, &user.Group_id, &user.Name)
	db.QueryRow("SELECT id FROM class WHERE id = '" + fmt.Sprint(user.Group_id) + "';").Scan(&group.Id)
	res, _ := db.Query("SELECT id, subject_id FROM subjects_groups WHERE group_id = '" + fmt.Sprint(group.Id) + "';")
	var sum, count, rate, rcount int
	var statsList []StatsForm
	for res.Next() {
		sum, count, rate, rcount = 0, 0, 0, 0
		res.Scan(&subject_group.Id, &subject_group.Subject_id)
		lessions, _ := db.Query("SELECT id, kind FROM lessions WHERE subject_group_id ='" + fmt.Sprint(subject_group.Id) + "';")
		for lessions.Next() {
			lessions.Scan(&lession.Id, &lession.Kind)
			rcount++
			err := db.QueryRow("SELECT grade, presence FROM lessions_users WHERE lession_id = '"+fmt.Sprint(lession.Id)+"' AND user_id = '"+fmt.Sprint(user.Id)+"';").Scan(&lession_user.Grade, &lession_user.Presence)
			if err == sql.ErrNoRows {
				continue
			}
			if lession_user.Presence == 1 {
				rate++
			}
			if lession.Kind != "Лекция" {
				sum += lession_user.Grade
				count++
			}
		}
		if count == 0 {
			count++
		}
		if rcount == 0 {
			rcount++
		}
		db.QueryRow("SELECT name FROM subjects WHERE id = '" + fmt.Sprint(subject_group.Subject_id) + "';").Scan(&subject.Name)
		statsList = append(statsList, StatsForm{subject.Name, float64(sum) / float64(count), rate * 100 / rcount})
	}

	t, err := template.ParseFiles("templates/stats.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "stats", MainStatsForm{user.Name, statsList})
}

func subjectf(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	res, err := db.Query("SELECT name, shortname FROM subjects")
	if err != nil {
		panic(err)
	}
	var table_list = []PairString{}
	for res.Next() {
		var cn = [2]string{}
		err = res.Scan(&cn[0], &cn[1])
		if err != nil {
			panic(err)
		}
		table_list = append(table_list, PairString{cn[0], cn[1]})
	}
	listStruct := ListForm{Username: userForm.Name, Type: "subject", Name: "Ваши предметы", Btn_name: "Выбрать предмет", List: table_list}

	t, err := template.ParseFiles("templates/list.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "list", listStruct)
}

func groupf(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var subject_id string
	res := db.QueryRow("SELECT id FROM subjects WHERE shortname = '" + vars["sub_name"] + "';")
	err = res.Scan(&subject_id)
	if err != nil {
		panic(err)
	}
	rows, err := db.Query("SELECT group_id FROM subjects_groups WHERE subject_id = '" + subject_id + "';")
	if err != nil {
		panic(err)
	}
	var group_id string
	var table_list = []PairString{}
	var cn [2]string
	for rows.Next() {
		err = rows.Scan(&group_id)
		if err != nil {
			panic(err)
		}
		res := db.QueryRow("SELECT name, shortname FROM class WHERE id = '" + group_id + "';")
		err = res.Scan(&cn[0], &cn[1])
		if err != nil {
			panic(err)
		}
		table_list = append(table_list, PairString{cn[0], cn[1]})
	}
	listStruct := ListForm{Username: userForm.Name, Type: "subject/" + vars["sub_name"], Name: "Группы", Btn_name: "Выбрать группу", List: table_list}

	t, err := template.ParseFiles("templates/list.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "list", listStruct)
}

func table(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var subject_id string
	db.QueryRow("SELECT id FROM subjects WHERE shortname = '" + vars["sub_name"] + "';").Scan(&subject_id)

	var group_id string
	db.QueryRow("SELECT id FROM class WHERE shortname = '" + vars["group_name"] + "';").Scan(&group_id)

	var subject_group_id string
	db.QueryRow("SELECT id FROM subjects_groups WHERE subject_id = '" + subject_id + "' AND group_id = '" + group_id + "';").Scan(&subject_group_id)

	if r.Method == "POST" {
		insert, err := db.Query(fmt.Sprintf("INSERT INTO lessions (subject_group_id, kind, date) VALUES ('%s', '%s','%s')", subject_group_id, r.FormValue("listGroupRadios"), r.FormValue("date")))
		if err != nil {
			panic(err)
		}
		defer insert.Close()

		http.Redirect(w, r, fmt.Sprintf("/subject/%s/%s", vars["sub_name"], vars["group_name"]), http.StatusSeeOther)
	} else {

		lessions, _ := db.Query("SELECT id, kind, date FROM lessions WHERE subject_group_id = '" + subject_group_id + "';")
		users, _ := db.Query("SELECT id, name FROM users WHERE group_id = '" + group_id + "';")

		var lessionList []Lession
		for lessions.Next() {
			lessions.Scan(&lession.Id, &lession.Kind, &lession.Date)
			lessionList = append(lessionList, Lession{Id: lession.Id, Kind: lession.Kind, Date: lession.Date})
		}
		var rowList []RowForm
		var sum, count, rate, rcount int

		for users.Next() {
			sum, count, rate, rcount = 0, 0, 0, 0
			users.Scan(&user.Id, &user.Name)
			var elemList []ElemForm
			for _, lession := range lessionList {
				rcount++
				err := db.QueryRow("SELECT * FROM lessions_users WHERE lession_id = '"+fmt.Sprint(lession.Id)+"' AND user_id = '"+fmt.Sprint(user.Id)+"';").Scan(&lession_user.Id, &lession_user.Lession_id, &lession_user.User_id, &lession_user.Presence, &lession_user.Grade)
				if err == sql.ErrNoRows {
					insert, err := db.Query("INSERT INTO lessions_users (lession_id, user_id, presence, grade) VALUES ('" + fmt.Sprint(lession.Id) + "', '" + fmt.Sprint(user.Id) + "', '0', '0')")
					if err != nil {
						panic(err)
					}
					defer insert.Close()
				}
				db.QueryRow("SELECT * FROM lessions_users WHERE lession_id = '"+fmt.Sprint(lession.Id)+"' AND user_id = '"+fmt.Sprint(user.Id)+"';").Scan(&lession_user.Id, &lession_user.Lession_id, &lession_user.User_id, &lession_user.Presence, &lession_user.Grade)
				elem := ElemForm{lession_user.Id, 1, fmt.Sprint(lession_user.Grade)}
				if lession_user.Presence == 1 {
					rate++
				}
				if lession.Kind == "Лекция" {
					if lession_user.Presence == 1 {
						elem.Grade = "+"
					} else {
						elem.Grade = "-"
					}
				} else {
					sum += lession_user.Grade
					count++
				}
				elemList = append(elemList, elem)
			}
			if count == 0 {
				count++
			}
			if rcount == 0 {
				rcount++
			}
			rowList = append(rowList, RowForm{user.Name, rate * 100 / rcount, float64(sum) / float64(count), elemList})
		}

		t, err := template.ParseFiles("templates/table.html", "templates/base.html")
		if err != nil {
			panic(err)
		}
		t.ExecuteTemplate(w, "table", MainTableForm{userForm.Name, vars["sub_name"], rowList, lessionList})
	}
}

func update_grade(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.QueryRow(fmt.Sprintf("SELECT lession_id, presence, grade FROM lessions_users WHERE id = '%s'", vars["lession_user_id"])).Scan(&lession_user.Lession_id, &lession_user.Presence, &lession_user.Grade)
	db.QueryRow(fmt.Sprintf("SELECT kind FROM lessions WHERE id = '%s'", fmt.Sprint(lession_user.Lession_id))).Scan(&lession.Kind)

	if r.Method == "POST" {
		if lession.Kind == "Лекция" {
			update, err := db.Query(fmt.Sprintf("UPDATE lessions_users SET presence = '%s', grade = '0' WHERE id = %s", r.FormValue("listG"), vars["lession_user_id"]))
			if err != nil {
				panic(err)
			}
			defer update.Close()
		} else {
			update, err := db.Query(fmt.Sprintf("UPDATE lessions_users SET presence = '%s', grade = '%s' WHERE id = %s", r.FormValue("listG"), r.FormValue("listGroup"), vars["lession_user_id"]))
			if err != nil {
				panic(err)
			}
			defer update.Close()
		}
		http.Redirect(w, r, fmt.Sprintf("/subject/%s/%s", vars["sub_name"], vars["group_name"]), http.StatusSeeOther)

	} else {

		flag := 1
		if lession.Kind == "Лекция" {
			flag = 0
		}
		db.QueryRow("SELECT name FROM class WHERE shortname = '" + vars["group_name"] + "';").Scan(&group.Name)

		t, err := template.ParseFiles("templates/update_grade.html", "templates/base.html")
		if err != nil {
			panic(err)
		}
		t.ExecuteTemplate(w, "update", UpdateForm{userForm.Name, group.Name, vars["group_name"], vars["sub_name"], flag})
	}
}

func admin(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	table_list := []PairString{{"users", "users"}, {"class", "class"}, {"subjects", "subjects"}, {"subjects_groups", "subjects_groups"},
		{"lessions", "lessions"}, {"lessions_users", "lessions_users"}}
	listStruct := ListForm{Username: userForm.Name, Type: "admin", Name: "Таблицы БД", Btn_name: "Выбрать таблицу", List: table_list}

	t, err := template.ParseFiles("templates/list.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "list", listStruct)
}

func admin_table(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var tableStruct TableForm
	tableStruct.Name = vars["table"]
	tableStruct.Username = userForm.Name

	res, err := db.Query(fmt.Sprintf("SELECT * FROM %s", vars["table"]))
	if err != nil {
		panic(err)
	}

	var contents = [][]string{}
	obj := model_map[vars["table"]]
	tableStruct.Titles = obj.get_labels()
	switch vars["table"] {
	case "users":
		for res.Next() {
			var cn = [8]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3], &cn[4], &cn[5], &cn[6], &cn[7])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	case "class":
		for res.Next() {
			var cn = [3]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	case "subjects":
		for res.Next() {
			var cn = [4]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	case "subjects_groups":
		for res.Next() {
			var cn = [3]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	case "lessions":
		for res.Next() {
			var cn = [4]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	case "lessions_users":
		for res.Next() {
			var cn = [5]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3], &cn[4])
			if err != nil {
				panic(err)
			}
			contents = append(contents, cn[:])
		}
	}

	tableStruct.Contents = contents

	t, err := template.ParseFiles("templates/admin_table.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	t.ExecuteTemplate(w, "admin_table", tableStruct)
}

func create(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	vars := mux.Vars(r)

	obj := model_map[vars["table"]]
	labels := obj.get_labels()[1:]

	if r.Method == "POST" {
		db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
		if err != nil {
			panic(err)
		}
		defer db.Close()

		query := "INSERT INTO " + vars["table"] + " ("
		for i := range labels {
			query += labels[i]
			if i+1 < len(labels) {
				query += ", "
			}
		}
		query += ") VALUES ("
		for i := range labels {
			query += "'" + r.FormValue(labels[i]) + "'"
			if i+1 < len(labels) {
				query += ", "
			}
		}
		query += ")"

		insert, err := db.Query(query)
		if err != nil {
			panic(err)
		}
		defer insert.Close()

		http.Redirect(w, r, fmt.Sprintf("/admin/%s", vars["table"]), http.StatusSeeOther)

	} else {

		var formS []CrudForm
		for i := 0; i < len(labels); i++ {
			fs := CrudForm{Form: labels[i], Value: ""}
			formS = append(formS, fs)
		}

		formStruct := CrudTableForm{Username: userForm.Name, Name: "Создать", Crud: formS, Type: vars["table"]}

		t, err := template.ParseFiles("templates/create.html", "templates/base.html")
		if err != nil {
			panic(err)
		}
		t.ExecuteTemplate(w, "create", formStruct)
	}
}

func update(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	vars := mux.Vars(r)
	obj := model_map[vars["table"]]
	labels := obj.get_labels()[1:]

	if r.Method == "POST" {

		query := "UPDATE " + vars["table"] + " SET "
		for i, el := range labels {
			query += el + " = '" + r.FormValue(el) + "'"
			if i+1 < len(labels) {
				query += ", "
			}
		}
		query += " WHERE id = '" + vars["id"] + "'"

		update, err := db.Query(query)
		if err != nil {
			panic(err)
		}
		defer update.Close()

		http.Redirect(w, r, fmt.Sprintf("/admin/%s", vars["table"]), http.StatusSeeOther)

	} else {

		res := db.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id = '%s'", vars["table"], vars["id"]))

		var content []string
		switch vars["table"] {
		case "users":
			var cn = [8]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3], &cn[4], &cn[5], &cn[6], &cn[7])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		case "class":
			var cn = [3]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		case "subjects":
			var cn = [4]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		case "subjects_groups":
			var cn = [3]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		case "lessions":
			var cn = [4]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		case "lessions_users":
			var cn = [5]string{}
			err = res.Scan(&cn[0], &cn[1], &cn[2], &cn[3], &cn[4])
			if err != nil {
				panic(err)
			}
			content = cn[:]
		}

		var formS []CrudForm
		for i, el := range labels {
			fs := CrudForm{Form: el, Value: content[i+1]}
			formS = append(formS, fs)
		}

		formStruct := CrudTableForm{Username: userForm.Name, Name: "Изменить", Crud: formS, Type: vars["table"]}

		t, err := template.ParseFiles("templates/create.html", "templates/base.html")
		if err != nil {
			panic(err)
		}
		t.ExecuteTemplate(w, "create", formStruct)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if !userForm.Auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	if userForm.Role == "student" {
		http.Redirect(w, r, "/stats/"+userForm.Login, http.StatusSeeOther)
	}
	if userForm.Role == "teacher" {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	vars := mux.Vars(r)
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	delete, err := db.Query(fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", vars["table"], vars["id"]))
	if err != nil {
		panic(err)
	}
	defer delete.Close()

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", vars["table"]), http.StatusSeeOther)
}

func login(w http.ResponseWriter, r *http.Request) {

	ses, err := cookieStore.Get(r, cookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	t, err := template.ParseFiles("templates/login.html", "templates/base.html")
	if err != nil {
		panic(err)
	}

	if r.Method == "POST" {
		login := r.FormValue("floatingInput")
		password := r.FormValue("floatingPassword")

		err = db.QueryRow(fmt.Sprintf("SELECT login FROM users WHERE login = '%s' AND password = '%s'", login, password)).Scan(&user.Login)
		if err == sql.ErrNoRows {
			t.ExecuteTemplate(w, "login", nil)
		} else {
			ses.Values[sesKeyLogin] = user.Login
			err = cookieStore.Save(r, w, ses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, "/subject", http.StatusSeeOther)
		}

	} else {
		login, ok := ses.Values[sesKeyLogin].(string)
		err = db.QueryRow(fmt.Sprintf("SELECT login FROM users WHERE login = '%s'", login)).Scan(&user.Login)
		if err == sql.ErrNoRows {
			t.ExecuteTemplate(w, "login", nil)
		} else if !ok {
			t.ExecuteTemplate(w, "login", nil)
		} else {
			http.Redirect(w, r, "/subject", http.StatusSeeOther)
		}
	}
}

func logout(w http.ResponseWriter, r *http.Request) {

	ses, err := cookieStore.Get(r, cookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ses.Values[sesKeyLogin] = "anonymous"
	err = cookieStore.Save(r, w, ses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func signup(w http.ResponseWriter, r *http.Request) {

	userForm := check_user(w, r)
	if userForm.Auth {
		http.Redirect(w, r, "/subject", http.StatusSeeOther)
	}

	t, err := template.ParseFiles("templates/signup.html", "templates/base.html")
	if err != nil {
		panic(err)
	}

	if r.Method == "POST" {
		db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
		if err != nil {
			panic(err)
		}
		defer db.Close()

		group_name := r.FormValue("group")
		name := r.FormValue("name")
		email := r.FormValue("email")
		login := r.FormValue("login")
		password := r.FormValue("password")

		if name == "" || email == "" || login == "" || password == "" || group_name == "" {
			http.Redirect(w, r, "/signup", http.StatusSeeOther)
		}

		var group Group
		db.QueryRow(fmt.Sprintf("SELECT id FROM class WHERE naem = '%s'", group_name)).Scan(&group.Id)

		insert, err := db.Query(fmt.Sprintf("INSERT INTO users (group_id, name, email, login, password, is_teacher, is_admin) VALUES ('%s', '%s','%s', '%s', '%s', '0', '0')", fmt.Sprint(group.Id), name, email, login, password))
		if err != nil {
			panic(err)
		}
		defer insert.Close()

		http.Redirect(w, r, "/login", http.StatusSeeOther)

	} else {
		arr := []PairString{{"Группа", "group"}, {"ФИО", "name"}, {"Email", "email"}, {"Логин", "login"}, {"Пароль", "password"}}
		t.ExecuteTemplate(w, "signup", arr)

	}
}

func handleRequest() {
	r := mux.NewRouter()
	r.HandleFunc("/signup", signup).Methods("GET", "POST")
	r.HandleFunc("/login", login).Methods("GET", "POST")
	r.HandleFunc("/logout", logout).Methods("GET")
	r.HandleFunc("/", index).Methods("GET")
	r.HandleFunc("/stats/{login}", stats).Methods("GET")
	r.HandleFunc("/subject", subjectf).Methods("GET")
	r.HandleFunc("/subject/{sub_name}", groupf).Methods("GET")
	r.HandleFunc("/subject/{sub_name}/{group_name}", table).Methods("GET", "POST")
	r.HandleFunc("/subject/{sub_name}/{group_name}/{lession_user_id}", update_grade).Methods("GET", "POST")
	r.HandleFunc("/admin", admin).Methods("GET")
	r.HandleFunc("/admin/{table}", admin_table).Methods("GET")
	r.HandleFunc("/admin/{table}/create", create).Methods("GET", "POST")
	r.HandleFunc("/admin/{table}/update/{id}", update).Methods("GET", "POST")
	r.HandleFunc("/admin/{table}/delete/{id}", delete).Methods("GET", "POST")

	http.Handle("/", r)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.ListenAndServe(":8080", nil)
}

func check_user(w http.ResponseWriter, r *http.Request) UserForm {
	ses, err := cookieStore.Get(r, cookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return UserForm{"", "", "", false}
	}

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/students_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	login, ok := ses.Values[sesKeyLogin].(string)
	var user User
	err = db.QueryRow(fmt.Sprintf("SELECT name, is_teacher, is_admin, login FROM users WHERE login = '%s'", login)).Scan(&user.Name, &user.Is_teacher, &user.Is_admin, &user.Login)
	if err == sql.ErrNoRows {
		return UserForm{"", "", "", false}
	} else if !ok {
		return UserForm{"", "", "", false}
	} else {
		userForm := UserForm{user.Name, user.Login, "", true}
		if user.Is_admin {
			userForm.Role = "admin"
		} else if user.Is_teacher {
			userForm.Role = "teacher"
		} else {
			userForm.Role = "student"
		}

		return userForm
	}
}

func main() {

	gob.Register(sesKey(0))

	handleRequest()
}
