package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	Id       int    `gorm:"primaryKey"`
	Username string `gorm:"not null"`
}

func GETfunc(conn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ormchan := make(chan User)

		prid := r.URL.Query().Get("productid")
		if prid == "" {
			prid = r.URL.Query().Get("productname")
			if prid == "" {
				http.Error(w, "", http.StatusBadRequest)
			}
		}
		prid1,errprid:=strconv.Atoi(prid)
		if errprid != nil{
			go getrequest(prid, ormchan, conn)
		}else{
			go getrequest(prid1, ormchan, conn)
		}
		user := <-ormchan

		resp := map[string]string{"Id": string(user.Id), "Username": string(user.Username)}
		jsonresp, jsonerr := json.Marshal(resp)
		if jsonerr != nil {
			http.Error(w, "", http.StatusBadRequest)
		}
		w.Write(jsonresp)
	}

}
func POSTfunc(conn *gorm.DB, trychan chan User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sgchan := make(chan bool)
		channel1 := make(chan User)
		var body map[string]string
		defer r.Body.Close()
		dg, readerr := io.ReadAll(r.Body)
		if readerr != nil {
			http.Error(w, "", http.StatusBadRequest)
		}
		errm := json.Unmarshal(dg, &body)
		if errm != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			fmt.Println("Unmarshal error:", errm)
			return
		}
		for key, value := range body {
			if key == "productname" {
				go postrequest(value, conn, trychan, sgchan)
				tf := <-sgchan
				if tf == true {
					go getrequest(value, channel1, conn)
					resppr := <-channel1
					respmap := map[string]interface{}{"Id": resppr.Id, "productname": resppr.Username}
					jsonresp, jsonerr := json.Marshal(respmap)
					if jsonerr != nil {
						http.Error(w, "", http.StatusBadRequest)
					}
					w.Write(jsonresp)
				}
			}
		}

	}
}
func PUTfunc(conn *gorm.DB, trychan chan User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		var id int
		var name string
		sgchan := make(chan bool)
		readaka, err12 := io.ReadAll(r.Body)
		if err12 != nil {

			http.Error(w, "Failed to read body", http.StatusBadRequest)
		}
		defer r.Body.Close()
		err := json.Unmarshal(readaka, &body)
		if err != nil {
			fmt.Println("Unmarshall error")
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
		} else {
			for key, value := range body {
				if key == "productid" {
					val, err := value.(float64)
					if err {
						fmt.Println("Convert error",err)
					}
					id = int(val)
				}
				if key == "productname" {
					val, okay := value.(string)
					if okay {
						name = val
					}
				}
			}
			go updaterequest(id, name, conn, trychan, sgchan)
			select {
			case tf := <-sgchan:
				if tf == true {
					fmt.Println("Product has been updated")
				} else {

				}
			}
		}
	}
}
func DELETEfunc(trychan chan User, conn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sgchan := make(chan bool)
		prid := r.URL.Query().Get("productid")
		if prid == "" {
			prid = r.URL.Query().Get("productname")
			if prid == "" {
				http.Error(w, "", http.StatusBadRequest)
			}
		}
		prid1,errprid:=strconv.Atoi(prid)
		if errprid != nil{
			go deleterequest(prid, conn, trychan, sgchan)
		}else{
			go deleterequest(prid1, conn, trychan, sgchan)
		}
	

		select{

		case tf:=<-sgchan:
			if tf == true{

				fmt.Println("Product has been deleted")
			}

		}
		
	}

}
func register(conn *gorm.DB, trychan chan User, wg *sync.WaitGroup) {
	http.HandleFunc("/get", GETfunc(conn))
	http.HandleFunc("/post", POSTfunc(conn, trychan))
	http.HandleFunc("/update", PUTfunc(conn, trychan))
	http.HandleFunc("/delete", DELETEfunc(trychan, conn))
	defer wg.Done()

}
func bdstart(connchan chan *gorm.DB) {
	bdinfo := "host=localhost user=bot password=bot dbname=bot port=5432 sslmode = disable"
	conn, err := gorm.Open(postgres.Open(bdinfo), &gorm.Config{})
	if err != nil {
		panic("Can't connect to postgres")
	}
	conn.AutoMigrate(&User{})
	fmt.Println("Database conn has been established")
	connchan <- conn

}
func serverstart() {
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Cant connect to server")
	}
}
func buildPOSTrequest(name string, respchan chan *http.Response) {
	client := http.Client{}
	body := map[string]string{"productname": name}
	jsonbody, err := json.Marshal(body)
	if err != nil {
		panic("Marshal Error")
	}
	readbody := bytes.NewBuffer(jsonbody)
	req, reqerr := http.NewRequest("POST", "http://localhost:8080/post", readbody)
	req.Header.Set("Content-Type", "application/json")
	if reqerr != nil {
		panic("Building POST1 request ERROR")
	}
	response, doerr := client.Do(req)
	if doerr != nil {
		panic("Cant send your request(")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		fmt.Printf("Response body: %s\n", body)
	}
	defer response.Body.Close()
	respchan <- response

}
func buildPUTrequest(id int, name string, respchan chan *http.Response) {
	client := http.Client{}
	body := map[string]interface{}{"productid": id, "productname": name}
	jsonbody, err := json.Marshal(body)
	if err != nil {
		panic("Marshal Error")
	}
	readbody := bytes.NewBuffer(jsonbody)
	req, reqerr := http.NewRequest("PUT", "http://localhost:8080/update", readbody)
	if reqerr != nil {
		panic("Building put request ERROR")
	}
	response, doerr := client.Do((req))
	if doerr != nil {
		panic("Cant send your request(")
	}
	respchan <- response

}
func buildGETrequest(id interface{}, respchan chan *http.Response) {
	client := http.Client{}
	params := url.Values{}

	var id1 string
	switch tp := id.(type) {
	case int:
		id1 = strconv.Itoa(tp)
	case string:
		id1 = tp
	default:
		fmt.Println("Unsupported type")
		return
	}
	if _, err1 := strconv.Atoi(id1); err1 != nil {
		params.Add("productid", id1)
	} else {
		params.Add("productname", id1)
	}
	mainadress := "http://localhost:8080/get"
	url := fmt.Sprintf("%s?%s", mainadress, params.Encode())
	req, reqerr := http.NewRequest("GET", url, nil)
	if reqerr != nil {
		panic(reqerr)
	}
	resp, doerr := client.Do(req)
	if doerr != nil {
		panic("Cant send your request(")
	}
	respchan <- resp
}
func buildDELETErequest(id interface{}, respchan chan *http.Response) {
	client := http.Client{}
	param := url.Values{}
	var id1 string
	switch v := id.(type) {
	case int:
		id1 = strconv.Itoa(v)
	case string:
		id1 = v
	default:
		fmt.Println("Unsupported type")
		return
	}
	if _, err := strconv.Atoi(id1); err != nil {
		param.Add("productname", id1)
	} else {

		param.Add("productid", id1)
	}

	mainadress := "http://localhost:8080/delete"
	url := fmt.Sprintf("%s?%s", mainadress, param.Encode())

	req, reqerr := http.NewRequest("DELETE", url, nil)
	if reqerr != nil {
		panic("Building Delete request ERROR")
	}
	resp, doerr := client.Do(req)
	if doerr != nil {
		panic("Can't send your request")
	}
	respchan <- resp
}

func getrequest(name interface{}, ormchan chan User, conn *gorm.DB) {
	var user User
	var par1 string
	switch tp := name.(type) {
	case string:
		par1 = tp
		err := conn.First(&user, "username = ?", par1).Error
		if err != nil {
			fmt.Println("Can't find anything suitable", err)
		}
	case int:
		intpar := int(tp)
		err := conn.First(&user, "id = ?", intpar).Error
		if err == gorm.ErrRecordNotFound {
			fmt.Println("Can't find anything suitable", err)
		} else {
			if err !=nil{
				fmt.Println("Error finding user:", err)
			}
		}
	}
	if user.Id != 0{
		fmt.Println(user)
	}
	ormchan <- user
}
func postrequest(name string, conn *gorm.DB, trychan chan User, postchan chan bool) {
	var user User

	go getrequest(name, trychan, conn)

	trypr := <-trychan
	if trypr.Id == 0 {
		user = User{Username: name}
		conn.Create(&user)
		fmt.Printf("Created")
		postchan <- true
	} else {
		fmt.Println("Product is already exists")
		postchan <- true

	}

}
func updaterequest(id int, name string, conn *gorm.DB, trychan chan User, putchan chan bool) {
	go getrequest(id, trychan, conn)
	trypr := <-trychan
	if trypr.Id != 0 {
		trypr.Username = name
		conn.Updates(&trypr)
		fmt.Println("Product has been successfully updated")
		putchan <- true
	} else {
		fmt.Println("Product for update doesn't exist")
		putchan <- false
	}
}
func deleterequest(id interface{}, conn *gorm.DB, trychan chan User, delchan chan bool) {

	go getrequest(id, trychan, conn)
	trypr := <-trychan
	if trypr.Id != 0 {
		conn.Delete(&trypr)
		delchan <- true
	} else {
		fmt.Println("Product for delete doesn't exist")
		delchan <- false
	}
}
func main() {
	connchan := make(chan *gorm.DB)
	respchan := make(chan *http.Response)
	trychan := make(chan User)
	var requesttype string
	var name string
	var id int
	var wg sync.WaitGroup
	go bdstart(connchan)

	connection := <-connchan
	connection.Debug()
	wg.Add(1)
	go register(connection, trychan, &wg)
	wg.Wait()

	go serverstart()
	fmt.Println("Server has been activated")
	for {
		fmt.Print("Type of Request:")
		fmt.Scanln(&requesttype)
		if strings.ToLower(requesttype) == "get"{
			fmt.Println("Redeem a name or id of your user to get")
			_, errscan := fmt.Scanln(&name)
			if errscan != nil {
				fmt.Println("The unexpected  error has appeared ", errscan)
			} else {
				go buildGETrequest(name, respchan)
				select {
				case resp := <-respchan:
					if resp.StatusCode != http.StatusOK {
						fmt.Println("Error in Get request")
	
					} else {
						fmt.Println(resp)
					}
				}
	
			}
	
		}
		if strings.ToLower(requesttype) == "post" {
			fmt.Println("Redeem a name  of your user to create")
			_, errscan := fmt.Scanln(&name)
			if errscan != nil {
				fmt.Println("The unexpected  error has appeared", errscan)
			} else {
	
				go buildPOSTrequest(name, respchan)
				select {
				case resp := <-respchan:
					if resp.StatusCode != http.StatusOK {
						fmt.Println("Error in POST request:", resp.Status)
					} else {
						fmt.Println("POST request successful")
					}
				}
	
			}
	
		}
		if strings.ToLower(requesttype) == "put" {
			fmt.Println("Redeem an id of your user to update")
			_, err1 := fmt.Scanln(&id)
			if err1 != nil {
				fmt.Println("The unexpected  error has appeared", err1)
			}
			fmt.Println("Redeem a name of your user to update")
			_, err2 := fmt.Scanln(&name)
			if err2 != nil {
				fmt.Println("The unexpected  error has appeared", err2)
			} else {
	
				go buildPUTrequest(id, name, respchan)
				select {
				case resp := <-respchan:
					if resp.StatusCode != http.StatusOK {
						fmt.Println("Error in put request", resp.Status)
					} else {
						fmt.Println(resp)
					}
				}}
	
			}
			if strings.ToLower(requesttype) == "delete" {
				fmt.Println("Redeem a name or id of your user to DELETE")
				_,err34:=fmt.Scanln(&name)
				if err34!=nil{
					fmt.Println("The unexp error",err34)
				}else{
					go buildDELETErequest(name, respchan)
					select {
					case resp := <-respchan:
						if resp.StatusCode != http.StatusOK {
							fmt.Println("Error in delete request")
						} else {
							fmt.Println(resp)
						}
	}
			}

			}
		}

	
}
