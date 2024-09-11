package main

import (
	todos "anthony/todos-go/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
)

var (
	firebaseConfig = firebase.Config{
		DatabaseURL:      "",
		ProjectID:        "",
		ServiceAccountID: "",
	}
	initalList = map[string][]todos.TodosItem{
		"Maze of Life & Bidcurement": {
			{Id: 1, Text: "Wireframe ideas for MOL", Completed: false},
			{Id: 2, Text: "Finish fleshing out a navigation interface", Completed: false},
			{Id: 3, Text: "Build a todo app app showcasing firebase auth in golang", Completed: false},
			{Id: 4, Text: "Build a frontend client in expo for mobile showcasing its web and mobile capabilities", Completed: false},
		},
		"Rivrb": {
			{Id: 1, Text: "Go on a walk", Completed: false},
			{Id: 2, Text: "Continue learning rust", Completed: false},
			{Id: 3, Text: "Build a todo app", Completed: false},
		},
		"Personal": {
			{Id: 1, Text: "Migrate old codebase in CRA CLI to Expo", Completed: false},
			{Id: 2, Text: "Make sure firebase does everything intended", Completed: false},
			{Id: 3, Text: "Implement XD design where things might have messed up", Completed: false},
		},
	}
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := godotenv.Load()
	if err != nil {
		logger.Error("No .env file found")
		os.Exit(1)
		return
	}

	addr, exists := os.LookupEnv("IP_ADDRESS")
	if !exists {
		logger.Error("Env variable NGROK_STATIC_DOMAIN missing")
		os.Exit(1)
		return
	}
	// Define routes
	mux := http.NewServeMux()

	// POST
	mux.HandleFunc("POST /todos", createATodoList)

	// GET
	mux.HandleFunc("GET /auth/{userUid}", getUser)
	mux.HandleFunc("GET /todos/{key}", getTodoList)
	mux.HandleFunc("GET /todos", geAllTodoLists)

	// PUT
	mux.HandleFunc("PUT /todos/{key}", updateTodoList)
	mux.HandleFunc("PUT /todos/item/{key}", addATodoItem)

	// DELETE
	mux.HandleFunc("DELETE /todos/{key}", deleteATodoList)
	mux.HandleFunc("DELETE /todos/item/{key}", deleteATodoItem)

	logger.Info("starting server")

	err = http.ListenAndServe(addr, mux)
	logger.Error(err.Error())
	os.Exit(1)
}

func createATodoList(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")

	if title == "" {
		http.Error(w, "Please pass a title in the query paramater", http.StatusBadRequest)
		return
	}

	var newTodosItems []todos.TodosItem

	err := json.NewDecoder(r.Body).Decode(&newTodosItems)

	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	initalList[title] = newTodosItems

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(initalList)
}

func geAllTodoLists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(initalList)
}

func getTodoList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("key")

	selectedList, ok := initalList[id]

	if !ok {
		http.Error(w, "List not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(selectedList)
}

func addATodoItem(w http.ResponseWriter, r *http.Request) {
	var newTodoItem todos.TodosItem
	key := r.PathValue("key")

	err := json.NewDecoder(r.Body).Decode(&newTodoItem)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	todoList, ok := initalList[key]
	if !ok {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	// Generate a new ID (this is a simple approach, you might want to use a more robust method)
	newID := int32(len(todoList) + 1)
	newTodoItem.Id = newID

	initalList[key] = append(todoList, newTodoItem)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(newTodoItem)
}

func updateTodoList(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	_, ok := initalList[key]
	if !ok {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	delete(initalList, key)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	w.WriteHeader(http.StatusNoContent)
}

func deleteATodoList(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	_, ok := initalList[key]
	if !ok {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	delete(initalList, key)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	w.WriteHeader(http.StatusNoContent)
}

func deleteATodoItem(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	parts := strings.Split(key, "/")

	// Check for
	if len(parts) != 2 {
		http.Error(w, "Invalid key format", http.StatusBadRequest)
		return
	}

	listKey := parts[0]
	itemIDStr := parts[1]

	todoList, ok := initalList[listKey]
	if !ok {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	var itemID int32
	_, err := fmt.Sscanf(itemIDStr, "%d", &itemID)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	for i, item := range todoList {
		if item.Id == itemID {
			initalList[listKey] = append(todoList[:i], todoList[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	w.Header().Set("Access-Control-Allow-Credentials", "true")

	http.Error(w, "Item not found", http.StatusNotFound)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, &firebaseConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	uid := r.PathValue("userUid")

	user := authClient.Users(ctx, uid)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	json.NewEncoder(w).Encode(&user)
}

// Middleware for auth
func authenticateRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Initalize variables
		ctx := context.Background()

		// Initalize app
		app, err := firebase.NewApp(ctx, &firebaseConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Initalize auth services
		client, err := app.Auth(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Grab the token from the request
		var idToken string

		authorization := r.Header.Get("Authorization")

		if len(strings.Split(authorization, " ")) > 1 {
			idToken = strings.Split(authorization, " ")[1]
		}

		// Check the token
		_, err = client.VerifyIDToken(ctx, idToken)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("User successfully authenticated")

		handler(w, r)
	}
}
