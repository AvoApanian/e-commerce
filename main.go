package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
	"strconv"

	"github.com/jackc/pgx/v4"
)

type Requete struct {
	Fichier string `json:"fichier"`
	Product_Id string `json:'"Product_Id"`
}

type StructSignLog struct {
	SignLog  string `json:"sign_log"`
	Mail     string `json:"mail"`
	Mdp      string `json:"mdp"`
	Remember bool   `json:"remember"`
	Theme    bool   `json:"theme"`
}

type StructForgetMdp struct {
	MailCode string `json:"mail_code"`
	Mail     string `json:"mail"`
	Code     int    `json:"code"`
}

type StructId struct {
	Id string `json:"product_id"`
}

func isHashing(mdp string) string {
	slice := []string{
		"&", "M", "9", "k", "]", "v", "?", "Z", ">", "!",
		"@", "y", "u", "i", "o", "p", "a", "s", "d", "f",
		"g", "h", "j", "k", "l", ";", "z", "x", "c", "v",
		"b", "n", "m", ",", ".", "/", "Q", "W", "E", "R",
		"T", "Y", "U", "I", "O", "P", "{", "}", "A", "S",
		"D", "F", "G", "H", "J", "K", "L", ":", "Z", "X",
		"C", "V", "B", "N", "M", "<", ">", "1", "2", "3",
		"4", "5", "6", "7", "8", "9", "0", "~", "!", "@",
		"#", "%", "^", "&", "*", "(", ")", "-", "=", "_",
		"+", "q", "w", "e", "r", "t", "y", "u", "i", "o",
		"p", "[", "]", "{", "}", "a", "s", "d", "f", "g",
		"h", "j", "k", "l", ";", "z", "x", "c", "v", "b",
		"n", "m", ",", ".", "?", "1", "2", "3", "4", "5",
		"6", "7", "8", "9", "0", "~", "!", "@", "#", "%",
		"^", "&", "*", "(", ")", "-", "=", "_", "+",
	}
	var hashed string
	for i := 0; i < len(mdp); i++ {
		char := string(mdp[i])

		for j, s := range slice {
			if s == char {
				hashed += fmt.Sprintf("%02d", j)
				break
			}
		}
	}
	return hashed
}

func isSign(w http.ResponseWriter, data StructSignLog, bdd *pgx.Conn) {
	mail := data.Mail
	mdp := data.Mdp
	remember := data.Remember
	theme := data.Theme

	if remember {
		fmt.Printf("Remember activé - Mail: %s, Theme: %v\n", mail, theme)
	}

	mailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	send_mail := regexp.MustCompile(mailRegex).MatchString(mail)
	send_mdp := false

	if len(mdp) >= 8 {
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		specialChars := "@$!%*?&"
		for _, char := range mdp {
			switch {
			case unicode.IsUpper(char):
				hasUpper = true
			case unicode.IsLower(char):
				hasLower = true
			case unicode.IsDigit(char):
				hasDigit = true
			case strings.ContainsRune(specialChars, char):
				hasSpecial = true
			}
		}
		send_mdp = hasUpper && hasLower && hasDigit && hasSpecial
	}

	if send_mail && send_mdp {
		mail_sel, err := Select(bdd, "user_bdd", mail)
		if err != nil {
			log.Println("Erreur dans Select:", err)
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		if mail_sel == mail {
			response := map[string]interface{}{"error": "Email est déjà utilisé"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		hash := isHashing(mdp)
		insert := `
			INSERT INTO user_bdd (mail, password, theme, bankName, cartNum, mm_aa, cvv)
			VALUES ($1, $2, false, 0, 0, 0, 0)
			RETURNING id
		`
		var id string
		err = bdd.QueryRow(context.Background(), insert, mail, hash).Scan(&id)
		if err != nil {
			fmt.Printf("Impossible d'insérer des données dans la BDD : %v\n", err)
			http.Error(w, "Erreur lors de l'insertion", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":   "success",
			"message":  "Inscription réussie",
			"id":       id,
			"remember": remember,
			"theme":    theme,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"mail":     send_mail,
		"mdp":      send_mdp,
		"remember": remember,
		"theme":    theme,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

func isLog(w http.ResponseWriter, data StructSignLog, bdd *pgx.Conn) {
	hash := isHashing(data.Mdp)

	if data.Remember {
		fmt.Printf("Remember activé - Mail: %s, Theme: %v\n", data.Mail, data.Theme)
	}

	var dbHash string
	var id string
	err := bdd.QueryRow(context.Background(),
		"SELECT id, password FROM user_bdd WHERE mail = $1", data.Mail).Scan(&id, &dbHash)

	if err != nil {
		if err == pgx.ErrNoRows {
			response := map[string]interface{}{
				"error": "Email ou mot de passe incorrect",
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}

		fmt.Printf("Erreur base de données: %v\n", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	if hash != dbHash {
		response := map[string]interface{}{
			"error": "Email ou mot de passe incorrect",
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"message":  "Connexion réussie",
		"id":       id,
        "mail":data.Mail,
		"remember": data.Remember,
		"theme":    data.Theme,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func generateRandomNumber() int {
	rand.Seed(time.Now().UnixNano())
	number := rand.Intn(900000) + 100000
	return number
}

var codeVerification int

func mailSend(structForget StructForgetMdp) {
	userMail := structForget.Mail
	codeVerification = generateRandomNumber()
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	myMail := "technova265@gmail.com"
	myMailPassword := "projet schollair"
	to := []string{userMail}
	subject := "Code Verification TechNova"
	body := fmt.Sprintf("Your verification code is: %d", codeVerification)
	auto := smtp.PlainAuth("", myMail, myMailPassword, smtpHost)
	message := []byte("Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auto, myMail, to, message)
	if err != nil {
		fmt.Printf("Erreur lors de l'envoi de l'email: %v\n", err)
		return
	}
	fmt.Println("Email envoyé avec succès!")
}

func code_chek(w http.ResponseWriter, structForget StructForgetMdp) {
	w.Header().Set("Content-Type", "application/json")
	user_code := structForget.Code
	iscode := false

	if user_code == codeVerification { 
		iscode = true
	}
	response := map[string]interface{}{
		"code": iscode,
	}
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(response)
}

func mobileData(bdd *pgx.Conn, w http.ResponseWriter) {
	rows, err := bdd.Query(context.Background(),
		`SELECT id, name, description, image, mesh, rating, price, count FROM product`,
	)
	if err != nil {
		http.Error(w, "Erreur lors de la requête SELECT", http.StatusInternalServerError)
		fmt.Printf("Erreur lors de la requête SELECT : %v\n", err)
		return
	}
	defer rows.Close()

	mobileNames := map[string]bool{
		"iphone16":         true,
		"samsung25":        true,
		"asusRog8":         true,
		"redmiNote14Pro+":  true,
	}

	var data []map[string]string

	for rows.Next() {
		var id, name, description, image, mesh string
		var rating, price, count int32

		err := rows.Scan(&id, &name, &description, &image, &mesh, &rating, &price, &count)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture d'une ligne : %v\n", err)
			continue
		}

		
		if !mobileNames[name] {
			continue
		}

		entry := map[string]string{
			"id":          id,
			"name":        name,
			"description": description,
			"image":       image,
			"mesh":        mesh,
			"rating":      strconv.Itoa(int(rating)),
			"price":       strconv.Itoa(int(price)),
			"count":       strconv.Itoa(int(count)),
		}

		data = append(data, entry)
	}

	response := map[string]interface{}{
		"reponse": data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getProductInfo(w http.ResponseWriter, r *http.Request, bdd *pgx.Conn) {
    w.Header().Set("Content-Type", "application/json")
    
    var prodId StructId
    err := json.NewDecoder(r.Body).Decode(&prodId)
    if err != nil {
        http.Error(w, "Données invalides", http.StatusBadRequest)
        return
    }

    query := `SELECT id, name, description, image, mesh, rating, price, count FROM product WHERE id = $1`
    var id, name, description, image, mesh string
    var rating, price, count int

    err = bdd.QueryRow(context.Background(), query, prodId.Id).Scan(&id, &name, &description, &image, &mesh, &rating, &price, &count)
    if err != nil {
        http.Error(w, "Produit non trouvé", http.StatusNotFound)
        fmt.Printf("Erreur SQL : %v\n", err)
        return
    }


    response := map[string]interface{}{
        "reponse": []map[string]interface{}{
            {
                "id":          id,
                "name":        name,
                "description": description,
                "image":       image,
                "mesh":        mesh,
                "rating":      rating,
                "price":       price,
                "count":       count,
            },
        },
    }

	w.WriteHeader(http.StatusOK)  
    json.NewEncoder(w).Encode(response)
}


func laptopData(w http.ResponseWriter, bdd *pgx.Conn) {
	rows, err := bdd.Query(context.Background(),
		`SELECT id, name, description, image, mesh, rating, price, count FROM product`,
	)
	if err != nil {
		http.Error(w, "Erreur lors de la requête SELECT", http.StatusInternalServerError)
		fmt.Printf("Erreur lors de la requête SELECT : %v\n", err)
		return
	}
	defer rows.Close()


	laptopNames := map[string]bool{
		"Macbook air M2":   true,
		"Dell XPS 13":        true,
		"Lenovo Legion 5 Pro": true,
	}

	var data []map[string]string

	for rows.Next() {
		var id, name, description, image, mesh string
		var rating, price, count int32

		err := rows.Scan(&id, &name, &description, &image, &mesh, &rating, &price, &count)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture d'une ligne : %v\n", err)
			continue
		}


		if !laptopNames[name] {
			continue
		}

		entry := map[string]string{
			"id":          id,
			"name":        name,
			"description": description,
			"image":       image,
			"mesh":        mesh,
			"rating":      strconv.Itoa(int(rating)),
			"price":       strconv.Itoa(int(price)),
			"count":       strconv.Itoa(int(count)),
		}

		data = append(data, entry)
	}

	response := map[string]interface{}{
		"reponse": data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}



func handler(w http.ResponseWriter, r *http.Request, bdd *pgx.Conn) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Erreur lecture du JSON", http.StatusBadRequest)
			return
		}

		var requete Requete
		if err := json.Unmarshal(body, &requete); err != nil {
			http.Error(w, "Error decode JSON (requete)", http.StatusBadRequest)
			return
		}

		if requete.Fichier == "index_html" {
			var loginData StructSignLog
			if err := json.Unmarshal(body, &loginData); err != nil {
				http.Error(w, "Error decode JSON (loginData)", http.StatusBadRequest)
				return
			}

			if loginData.SignLog == "sign" {
				isSign(w, loginData, bdd)
			} else if loginData.SignLog == "login" {
				isLog(w, loginData, bdd)
			} else {
				http.Error(w, "Action inconnue", http.StatusBadRequest)
			}
		} else if requete.Fichier == "forget_html" {
			var sturctForget StructForgetMdp
			err := json.Unmarshal(body, &sturctForget)
			if err != nil {
				fmt.Printf("Erreur decodage JSON (structForget):%s:", err)
				http.Error(w, "Erreur decodage JSON (structForget)", http.StatusBadRequest)
				return
			}

			if sturctForget.MailCode == "mail" {
				mailSend(sturctForget)
			} else if sturctForget.MailCode == "code" {
				code_chek(w, sturctForget)
			}
		}  else if requete.Fichier == "mobile_html" {
			if requete.Product_Id != "" {
				fmt.Println("Produit selectionne :", requete.Product_Id)
				getProductInfo(w, r, bdd)
			} else {
				mobileData(bdd, w)
			}
		}else if requete.Fichier == "laptop_html" {
			if requete.Product_Id != "" {
				fmt.Printf("Produit selectionne %v",requete.Product_Id)
				getProductInfo(w,r,bdd)
			}else {
				laptopData(w,bdd)
			}
		}
		
	}
}



func main() {
	bdd, err := pgx.Connect(context.Background(),
		"postgres://postgres:eshuKluxdb@127.0.0.1:5432/amazone")
	if err != nil {
		log.Fatalf("Échec connexion BDD: %v", err)
	}
	defer bdd.Close(context.Background())

	_, err = bdd.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS user_bdd (
			id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
			mail VARCHAR(200) UNIQUE,
			password VARCHAR(200),
			theme BOOLEAN DEFAULT false,
			bankName INTEGER DEFAULT 0,
			cartNum INTEGER DEFAULT 0,
			mm_aa INTEGER DEFAULT 0,
			cvv INTEGER DEFAULT 0
		)`)
	if err != nil {
		log.Fatalf("Échec création table: %v", err)
	}

	_,product_err := bdd.Exec(context.Background(),
		`
			CREATE TABLE IF NOT EXISTS product (
				id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
				name VARCHAR(200) UNIQUE,
				description VARCHAR(300),
				image VARCHAR(200) UNIQUE,
				mesh VARCHAR(200) UNIQUE,
				rating INTEGER,
				price INTEGER,
				count INTEGER
			)
		`,
	)

	if product_err != nil {
		fmt.Printf("erreur de creation/use de la table product\n%v\n",product_err)
		return
	}

	// changeURl(bdd,"iphone16","https://github.com/AvoApanian/e-commerce/blob/main/3d/iphone16.fbx")
	// changeURl(bdd,"samsung25","https://github.com/AvoApanian/e-commerce/blob/main/3d/samsong%20galaxy%202515.fbx")
	// changeURl(bdd,"asusRog8","https://github.com/AvoApanian/e-commerce/blob/main/3d/ASUS_ROG_Phone_8.fbx")
	// changeURl(bdd,"redmiNote14Pro+","https://github.com/AvoApanian/e-commerce/blob/main/3d/Redmi_Not_%2014_Pro%2B.fbx")
	// changeURl(bdd,"Lenovo Legion 5 Pro","https://github.com/AvoApanian/e-commerce/blob/main/3d/%20Lenovo%20Legion%205%20Pro%20(windows).fbx")
	// changeURl(bdd,"Dell XPS 13","https://github.com/AvoApanian/e-commerce/blob/main/3d/Dell_XPS_13(windows).fbx")
	// changeURl(bdd,"Macbook air M2","https://github.com/AvoApanian/e-commerce/blob/main/3d/MacBook%20Air%20M2d.fbx")


	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "delete":
			DeleteAllUsers(bdd)
			return
		case "print/user_bd":
			PrintAllUsers(bdd)
			return
		case "print/product":
			PrintAllProduct(bdd)
		case "select":
			Select(bdd, "user_bdd", "*")
		default:
			fmt.Println("Usage: go run . [list|clean]")
			return
		}
	}

	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, bdd)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Serveur démarré sur :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}