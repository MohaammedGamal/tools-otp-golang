package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/sijms/go-ora/v2"
)

var connectionDetails = struct {
	Connections map[string]string
	Selected    string
}{Connections: make(map[string]string)}

const connectionFile = "connections.json"
const adminPassword = "securePassword123" // Replace with a more secure password

func main() {
	loadConnections()
	http.HandleFunc("/", queryPageHandler) // Default page is now the query page
	http.HandleFunc("/admin", adminPageHandler)
	http.HandleFunc("/save", saveDetailsHandler)
	http.HandleFunc("/fetch", fetchResultsHandler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConnections() {
	if _, err := os.Stat(connectionFile); err == nil {
		data, err := ioutil.ReadFile(connectionFile)
		if err != nil {
			log.Println("Error reading connections file:", err)
			return
		}
		if err := json.Unmarshal(data, &connectionDetails.Connections); err != nil {
			log.Println("Error parsing connections file:", err)
		}
	}
}

func saveConnections() {
	data, err := json.MarshalIndent(connectionDetails.Connections, "", "  ")
	if err != nil {
		log.Println("Error saving connections file:", err)
		return
	}
	if err := ioutil.WriteFile(connectionFile, data, 0644); err != nil {
		log.Println("Error writing connections file:", err)
	}
}

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		password := r.URL.Query().Get("password")
		if password != adminPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Connection Details</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f9f9f9;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background: #ffffff;
            border-radius: 10px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
        }
        h1 {
            text-align: center;
            color: #333333;
        }
        form {
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        label {
            font-size: 16px;
            font-weight: bold;
            color: #555555;
        }
        input[type="text"] {
            padding: 10px;
            border: 1px solid #dddddd;
            border-radius: 5px;
            font-size: 14px;
        }
        input[type="submit"] {
            padding: 10px;
            background-color: #007BFF;
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            cursor: pointer;
            transition: background-color 0.3s;
        }
        input[type="submit"]:hover {
            background-color: #0056b3;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Enter Connection Details</h1>
        <form action="/save" method="POST">
            <label for="name">Connection Name:</label>
            <input type="text" id="name" name="name" required>
            
            <label for="dsn">Oracle DSN:</label>
            <input type="text" id="dsn" name="dsn" required>
            
            <input type="submit" value="Save">
        </form>
    </div>
</body>
</html>
`
		template.Must(template.New("adminPage").Parse(tmpl)).Execute(w, nil)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func saveDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	dsn := r.FormValue("dsn")
	if name == "" || dsn == "" {
		http.Error(w, "Name and DSN are required", http.StatusBadRequest)
		return
	}

	connectionDetails.Connections[name] = dsn
	saveConnections()
	log.Printf("Saved connection: %s = %s", name, dsn)
	connectionDetails.Selected = name

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func queryPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Query Page</title>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/flatpickr/4.6.9/flatpickr.min.js"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/flatpickr/4.6.9/flatpickr.min.css" rel="stylesheet">
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f4f4f9;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #ffffff;
            border-radius: 10px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
        }
        h1 {
            text-align: center;
            color: #333333;
        }
        form {
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        label {
            font-size: 16px;
            font-weight: bold;
            color: #555555;
        }
        select, input[type="text"], input[type="checkbox"], input[type="submit"] {
            padding: 10px;
            border: 1px solid #dddddd;
            border-radius: 5px;
            font-size: 14px;
        }
        select, input[type="text"] {
            width: 100%;
        }
        input[type="submit"] {
            background-color: #28a745;
            color: white;
            border: none;
            cursor: pointer;
            transition: background-color 0.3s;
        }
        input[type="submit"]:hover {
            background-color: #218838;
        }
        .checkbox-label {
            display: flex;
            align-items: center;
            gap: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Select Database and Query Data</h1>
        <form action="/fetch" method="POST">
            <label for="database">Select Database:</label>
            <select id="database" name="database">
                {{range $key, $value := .Connections}}
                <option value="{{$key}}" {{if eq $.Selected $key}}selected{{end}}>{{$key}}</option>
                {{end}}
            </select>
            
            <label for="value">Enter Value:</label>
            <input type="text" id="value" name="value">
            
            <div class="checkbox-label">
                <input type="checkbox" id="executeWithoutValue" name="executeWithoutValue">
                <label for="executeWithoutValue">Execute without value</label>
            </div>
            
            <input type="submit" value="Fetch Results">
        </form>
    </div>
</body>
</html>
`
	template.Must(template.New("queryPage").Parse(tmpl)).Execute(w, connectionDetails)
}

func fetchResultsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbName := r.FormValue("database")
	value := r.FormValue("value")
	executeWithoutValue := r.FormValue("executeWithoutValue") == "on"

	dsn, ok := connectionDetails.Connections[dbName]
	if !ok {
		http.Error(w, "Selected database not found", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		log.Println("Database connection error:", err)
		return
	}
	defer db.Close()

	var query string
	var rows *sql.Rows
	if executeWithoutValue {
		query = "SELECT * FROM SMS.SMS ORDER BY ID DESC FETCH FIRST 20 ROWS ONLY"
		rows, err = db.Query(query)
	} else {
		query = "SELECT * FROM SMS.SMS WHERE MOBILE = :1"
		rows, err = db.Query(query, value)
	}

	if err != nil {
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		log.Println("Query execution error:", err)
		return
	}
	defer rows.Close()

	type Result struct {
		Column1 string
		Column2 string
		Column3 string
	}

	var results []Result
	for rows.Next() {
		var res Result
		if err := rows.Scan(&res.Column1, &res.Column2, &res.Column3); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		results = append(results, res)
	}

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Query Results</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f4f4f9;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 900px;
            margin: 50px auto;
            padding: 20px;
            background: #ffffff;
            border-radius: 10px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
        }
        h1 {
            text-align: center;
            color: #333333;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        table, th, td {
            border: 1px solid #dddddd;
        }
        th, td {
            text-align: left;
            padding: 10px;
        }
        th {
            background-color: #007BFF;
            color: white;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        tr:hover {
            background-color: #f1f1f1;
        }
        a {
            display: inline-block;
            margin-top: 20px;
            padding: 10px 15px;
            background-color: #007BFF;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            font-size: 14px;
            transition: background-color 0.3s;
        }
        a:hover {
            background-color: #0056b3;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Results</h1>
        <table>
            <tr>
                <th>Column1</th>
                <th>Column2</th>
                <th>Column3</th>
            </tr>
            {{range .}}
            <tr>
                <td>{{.Column1}}</td>
                <td>{{.Column2}}</td>
                <td>{{.Column3}}</td>
            </tr>
            {{end}}
        </table>
        <a href="/">Back</a>
    </div>
</body>
</html>
`
	template.Must(template.New("resultsPage").Parse(tmpl)).Execute(w, results)
}
