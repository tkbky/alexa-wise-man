package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	alexa "github.com/mikeflynn/go-alexa/skillserver"
)

var quotes = []string{
	"Only I can change my life. No one can do it for me.",
	"Life is 10% what happens to you and 90% how you react to it.",
	"Optimism is the faith that leads to achievement. Nothing can be done without hope and confidence.",
	"Good, better, best. Never let it rest. 'Til your good is better and your better is best.",
	"Our greatest weakness lies in giving up. The most certain way to succeed is always to try just one more time.",
	"Always do your best. What you plant now, you will harvest later.",
	"It always seems impossible until it's done.",
	"With the new day comes new strength and new thoughts.",
	"It does not matter how slowly you go as long as you do not stop.",
}

var applications = map[string]interface{}{
	"/echo/quotes": alexa.EchoApplication{
		AppID:   os.Getenv("ALEXA_SKILL_APP_ID"),
		Handler: QuotesHandler,
	},
}

var db *sqlx.DB

// Quote that is good needs no explain
type Quote struct {
	ID        string `db:"id"`
	Content   string `db:"content"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

var schema = `
CREATE TABLE IF NOT EXISTS quotes (
	id SERIAL NOT NULL PRIMARY KEY,
	content text NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);`

func main() {
	var err error

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	db, err = sqlx.Open("postgres", os.Getenv("DATABASE_URL"))

	if err != nil {
		log.Fatal(err)
	}

	db.MustExec(schema)

	if err != nil {
		log.Fatal(err)
	}

	var count int

	err = db.Get(&count, "SELECT COUNT(id) FROM quotes")

	if err != nil {
		log.Fatal(err)
	}

	if count <= 0 {
		seedQuotes()
	}

	rand.Seed(time.Now().UTC().UnixNano())
	alexa.Run(applications, port)
}

func seedQuotes() {
	tx := db.MustBegin()

	for _, quote := range quotes {
		tx.MustExec("INSERT INTO quotes(content) VALUES($1)", quote)
	}

	tx.Commit()
}

// QuotesHandler tells some wise quotes
func QuotesHandler(w http.ResponseWriter, r *http.Request) {
	echoReq := r.Context().Value("echoRequest").(*alexa.EchoRequest)

	switch echoReq.GetRequestType() {
	case "LaunchRequest":
		echoResp := launchResponse()
		handleResponse(w, echoResp)
	case "IntentRequest":
		switch echoReq.GetIntentName() {
		case "TellAQuote":
			echoResp := quoteResponse(echoReq)
			handleResponse(w, echoResp)
		case "HelpReply":
			echoResp := helpReply(echoReq)
			handleResponse(w, echoResp)
		case "AMAZON.HelpIntent":
			echoResp := helpResponse(echoReq)
			handleResponse(w, echoResp)
		default:
			echoResp := unknownResponse()
			handleResponse(w, echoResp)
		}
	}
}

func unknownResponse() *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech("I'm sorry, I didn't get that. Can you say that again?").EndSession(false)
}

func launchResponse() *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech("Hi, I'm Wise Man. Ask me for a quote.").EndSession(false)
}

func helpResponse(echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech("Just ask me for some inspirational quote by saying \"Inspire me with a quote\".").EndSession(false)
}

// Yeses are different words for acknowledgement
var Yeses = map[string]bool{
	"yes":  true,
	"sure": true,
}

func helpReply(echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	want, err := echoReq.GetSlotValue("Want")

	if err != nil {
		return unknownResponse()
	}

	_, ok := Yeses[strings.ToLower(want)]

	if ok {
		return quoteResponse(echoReq)
	}

	return alexa.NewEchoResponse().OutputSpeech("Alright, have a nice day!").EndSession(true)
}

func quoteResponse(echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	var content string

	err := db.Get(&content, "SELECT content FROM quotes WHERE quotes.id = $1", rand.Intn(len(quotes)))

	if err != nil {
		log.Fatal(err)
	}

	return alexa.NewEchoResponse().OutputSpeech(content).EndSession(true)
}

func handleResponse(w http.ResponseWriter, echoResp *alexa.EchoResponse) {
	json, _ := echoResp.String()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Write(json)
}
