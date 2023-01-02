package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/term"

	gogpt "github.com/sashabaranov/go-gpt3"
)

var (
	host     = flag.String("H", "gateway01.us-west-2.prod.aws.tidbcloud.com", "Host")
	port     = flag.Int("P", 4000, "Port")
	user     = flag.String("u", "4A7D3bbkQWsWSEH.guest", "user")
	password = flag.String("p", "11111111", "password")
	database = flag.String("D", "gharchive_dev", "database")
	key      = flag.String("key", "", "Your OpenAI API key")

	verbose = flag.Bool("verbose", false, "output full prompt")
)

func openDB(user, password, host string, port int, database string) *sql.DB {
	mysql.RegisterTLSConfig("tidb", &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: host,
	})

	dsn := fmt.Sprintf("%s@tcp(%s:%d)/%s?tls=tidb", strings.Join([]string{user, password}, ":"),
		host, port, database)
	db, err := sql.Open("mysql", dsn)
	panicErr(err)
	return db
}

func panicErr(err error) {
	if err == nil {
		return
	}

	panic(err.Error())
}

func buildRequest(prompt string) gogpt.CompletionRequest {
	req := gogpt.CompletionRequest{
		Model: "text-davinci-003",
		// Model:            "code-davinci-002",
		MaxTokens:        256,
		Prompt:           prompt,
		Temperature:      0,
		Stop:             []string{"#", "-", ";", "\n\n"},
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		BestOf:           1,
	}
	return req
}

func buildTablePrefix(db *sql.DB) string {
	tables, err := db.Query("show tables;")
	panicErr(err)
	defer tables.Close()

	var tableNames []string
	for tables.Next() {
		var tableName string
		err := tables.Scan(&tableName)
		panicErr(err)
		tableNames = append(tableNames, tableName)
	}

	var tablePrefix string
	tablePrefix = "# MySQL table\n"
	for _, tableName := range tableNames {
		s := fmt.Sprintf(`SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';`, *database, tableName)
		columns, err := db.Query(s)
		panicErr(err)
		defer columns.Close()

		var columnNames []string
		for columns.Next() {
			var columnName string
			err := columns.Scan(&columnName)
			panicErr(err)
			columnNames = append(columnNames, columnName)
		}

		tablePrefix += fmt.Sprintf("# Table %s, columns = [%s]\n", tableName, strings.Join(columnNames, ", "))
	}

	tablePrefix += `
# only output SQL 
# if no SQL can be generated, output "No SQL Generated"

`

	return tablePrefix
}

func main() {
	flag.Parse()

	c := gogpt.NewClient(*key)
	ctx := context.Background()

	db := openDB(*user, *password, *host, *port, *database)
	defer db.Close()

	tablePrefix := buildTablePrefix(db)

	oldState, err := term.MakeRaw(0)
	panicErr(err)

	defer term.Restore(0, oldState)
	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}
	term := term.NewTerminal(screen, "")
	term.SetPrompt(string(term.Escape.Red) + "prompt> " + string(term.Escape.Reset))

	for {
		prompt, err := term.ReadLine()
		if err == io.EOF {
			return
		}
		panicErr(err)

		if prompt == "" {
			continue
		}

		s := fmt.Sprintf("%s-- %s\n", tablePrefix, prompt)
		if *verbose {
			println(s)
		}

		req := buildRequest(s)
		resp, err := c.CreateCompletion(ctx, req)
		panicErr(err)

		fmt.Fprintln(term, resp.Choices[0].Text)
	}
}
