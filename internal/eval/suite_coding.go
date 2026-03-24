package eval

import "time"

// LoadCodingSuite returns 18 coding evaluation test cases covering
// code review, error handling, concurrency, API design, and testing.
func LoadCodingSuite() []TestCase {
	return []TestCase{
		// --- Code Review (4 cases) ---
		{
			ID:         "coding-001",
			Name:       "Nil pointer dereference in Go",
			Category:   CatCoding,
			Difficulty: Easy,
			Input: `Review this Go code:
func getUser(id string) *User {
    users := loadUsers()
    return users[id]
}
func handler(w http.ResponseWriter, r *http.Request) {
    user := getUser(r.URL.Query().Get("id"))
    fmt.Fprintf(w, "Hello %s", user.Name)
}`,
			Expected:  []string{"nil", "check", "panic", "pointer"},
			Forbidden: []string{"looks good", "no issues"},
			Tags:      []string{"code-review"},
			Timeout:   30 * time.Second,
		},
		{
			ID:         "coding-002",
			Name:       "Resource leak in Go",
			Category:   CatCoding,
			Difficulty: Easy,
			Input: `Review this Go code:
func readFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    data, err := io.ReadAll(f)
    if err != nil {
        return nil, err
    }
    return data, nil
}`,
			Expected:  []string{"close", "defer", "leak", "resource"},
			Forbidden: []string{"correct", "no issues"},
			Tags:      []string{"code-review"},
			Timeout:   30 * time.Second,
		},
		{
			ID:         "coding-003",
			Name:       "SQL injection vulnerability",
			Category:   CatCoding,
			Difficulty: Medium,
			Input: `Review this Go code:
func findUser(db *sql.DB, name string) (*User, error) {
    query := "SELECT id, name, email FROM users WHERE name = '" + name + "'"
    row := db.QueryRow(query)
    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email)
    return &u, err
}`,
			Expected:  []string{"SQL injection", "parameterized", "placeholder", "prepared"},
			Forbidden: []string{"safe", "no issues"},
			Tags:      []string{"code-review"},
			Timeout:   30 * time.Second,
		},
		{
			ID:         "coding-004",
			Name:       "Goroutine leak with unbuffered channel",
			Category:   CatCoding,
			Difficulty: Hard,
			Input: `Review this Go code:
func fetchAll(urls []string) []string {
    results := make(chan string)
    for _, url := range urls {
        go func(u string) {
            resp, err := http.Get(u)
            if err != nil {
                return
            }
            defer resp.Body.Close()
            body, _ := io.ReadAll(resp.Body)
            results <- string(body)
        }(url)
    }
    var out []string
    for i := 0; i < len(urls); i++ {
        out = append(out, <-results)
    }
    return out
}`,
			Expected: []string{"goroutine", "leak", "error", "channel", "block"},
			Tags:     []string{"code-review"},
			Timeout:  30 * time.Second,
		},

		// --- Error Handling (4 cases) ---
		{
			ID:         "coding-005",
			Name:       "Swallowed error in Go",
			Category:   CatCoding,
			Difficulty: Easy,
			Input: `Review the error handling:
func processItems(items []Item) {
    for _, item := range items {
        result, _ := transform(item)
        save(result)
    }
}`,
			Expected:  []string{"error", "ignore", "handle", "check"},
			Forbidden: []string{"correct", "fine"},
			Tags:      []string{"error-handling"},
			Timeout:   30 * time.Second,
		},
		{
			ID:         "coding-006",
			Name:       "Error wrapping best practice",
			Category:   CatCoding,
			Difficulty: Medium,
			Input: `How should this Go function handle and return errors?
func CreateOrder(ctx context.Context, order Order) error {
    if err := validateOrder(order); err != nil {
        return err
    }
    if err := db.InsertOrder(ctx, order); err != nil {
        return err
    }
    if err := notifyUser(ctx, order.UserID); err != nil {
        return err
    }
    return nil
}`,
			Expected: []string{"fmt.Errorf", "%w", "wrap", "context", "stack"},
			Tags:     []string{"error-handling"},
			Timeout:  30 * time.Second,
		},
		{
			ID:         "coding-007",
			Name:       "Sentinel errors vs error types",
			Category:   CatCoding,
			Difficulty: Medium,
			Input:      "When should I use sentinel errors (var ErrNotFound = errors.New(...)) versus custom error types in Go? Give examples of each.",
			Expected:   []string{"sentinel", "errors.Is", "errors.As", "type", "interface"},
			Tags:       []string{"error-handling"},
			Timeout:    30 * time.Second,
		},
		{
			ID:         "coding-008",
			Name:       "Panic recovery pattern",
			Category:   CatCoding,
			Difficulty: Hard,
			Input:      "Write a Go middleware that recovers from panics in HTTP handlers, logs the stack trace, and returns a 500 response. Explain when this is appropriate.",
			Expected:   []string{"recover", "defer", "500", "stack", "middleware"},
			Tags:       []string{"error-handling"},
			Timeout:    30 * time.Second,
		},

		// --- Concurrency (4 cases) ---
		{
			ID:         "coding-009",
			Name:       "Race condition on shared map",
			Category:   CatCoding,
			Difficulty: Easy,
			Input: `Identify the concurrency issue:
var cache = make(map[string]string)

func SetCache(key, value string) {
    cache[key] = value
}

func GetCache(key string) string {
    return cache[key]
}

// Called from multiple goroutines`,
			Expected: []string{"race", "mutex", "sync.Map", "concurrent"},
			Tags:     []string{"concurrency"},
			Timeout:  30 * time.Second,
		},
		{
			ID:         "coding-010",
			Name:       "Worker pool implementation",
			Category:   CatCoding,
			Difficulty: Medium,
			Input:      "Implement a Go worker pool that processes jobs from a channel with N concurrent workers, graceful shutdown on context cancellation, and error collection.",
			Expected:   []string{"goroutine", "channel", "context", "WaitGroup", "worker"},
			Tags:       []string{"concurrency"},
			Timeout:    30 * time.Second,
		},
		{
			ID:         "coding-011",
			Name:       "Deadlock detection",
			Category:   CatCoding,
			Difficulty: Hard,
			Input: `Find the deadlock:
var mu1 sync.Mutex
var mu2 sync.Mutex

func transferA() {
    mu1.Lock()
    defer mu1.Unlock()
    time.Sleep(time.Millisecond)
    mu2.Lock()
    defer mu2.Unlock()
    // transfer logic
}

func transferB() {
    mu2.Lock()
    defer mu2.Unlock()
    time.Sleep(time.Millisecond)
    mu1.Lock()
    defer mu1.Unlock()
    // transfer logic
}`,
			Expected: []string{"deadlock", "lock order", "consistent", "mu1", "mu2"},
			Tags:     []string{"concurrency"},
			Timeout:  30 * time.Second,
		},
		{
			ID:         "coding-012",
			Name:       "Context propagation",
			Category:   CatCoding,
			Difficulty: Medium,
			Input:      "Explain how to properly propagate context.Context through a Go application for cancellation, timeouts, and tracing. Show the anti-patterns to avoid.",
			Expected:   []string{"context.Context", "first parameter", "Background", "WithCancel", "WithTimeout"},
			Forbidden:  []string{"global variable"},
			Tags:       []string{"concurrency"},
			Timeout:    30 * time.Second,
		},

		// --- API Design (3 cases) ---
		{
			ID:         "coding-013",
			Name:       "REST API anti-patterns",
			Category:   CatCoding,
			Difficulty: Easy,
			Input: `Identify anti-patterns in this API:
POST /api/getUser
POST /api/deleteUser
GET  /api/users/create?name=John
GET  /api/updateUser/123?name=Jane`,
			Expected: []string{"verb", "GET", "POST", "DELETE", "noun", "HTTP method"},
			Tags:     []string{"api-design"},
			Timeout:  30 * time.Second,
		},
		{
			ID:         "coding-014",
			Name:       "API pagination design",
			Category:   CatCoding,
			Difficulty: Medium,
			Input:      "Design pagination for a REST API that lists items from a large dataset (10M+ records). Compare offset-based, cursor-based, and keyset pagination. Which is best for mobile clients?",
			Expected:   []string{"cursor", "offset", "keyset", "next", "performance"},
			Tags:       []string{"api-design"},
			Timeout:    30 * time.Second,
		},
		{
			ID:         "coding-015",
			Name:       "API versioning strategy",
			Category:   CatCoding,
			Difficulty: Medium,
			Input:      "Our public API is used by 500+ external clients. We need to make breaking changes. Compare URL path versioning (/v2/), header versioning, and query param versioning.",
			Expected:   []string{"version", "backward", "deprecat", "migration"},
			Tags:       []string{"api-design"},
			Timeout:    30 * time.Second,
		},

		// --- Testing (3 cases) ---
		{
			ID:         "coding-016",
			Name:       "Missing test cases identification",
			Category:   CatCoding,
			Difficulty: Medium,
			Input: `This function calculates shipping cost. What test cases are missing?
func ShippingCost(weight float64, expedited bool, country string) (float64, error) {
    if weight <= 0 { return 0, ErrInvalidWeight }
    if country == "" { return 0, ErrInvalidCountry }
    base := weight * 2.5
    if expedited { base *= 1.5 }
    if country != "US" { base += 15.0 }
    return base, nil
}
Existing tests: TestShippingCost_Normal, TestShippingCost_Expedited`,
			Expected: []string{"boundary", "negative", "zero", "international", "country", "error"},
			Tags:     []string{"testing"},
			Timeout:  30 * time.Second,
		},
		{
			ID:         "coding-017",
			Name:       "Table-driven test pattern",
			Category:   CatCoding,
			Difficulty: Easy,
			Input:      "Convert these individual Go test functions into a table-driven test:\nfunc TestAdd_Positive() { assert(Add(2,3), 5) }\nfunc TestAdd_Negative() { assert(Add(-1,-2), -3) }\nfunc TestAdd_Zero() { assert(Add(0,0), 0) }\nfunc TestAdd_Mixed() { assert(Add(-1,1), 0) }",
			Expected:   []string{"[]struct", "t.Run", "test case", "range", "tt."},
			Tags:       []string{"testing"},
			Timeout:    30 * time.Second,
		},
		{
			ID:         "coding-018",
			Name:       "Mocking external dependencies",
			Category:   CatCoding,
			Difficulty: Hard,
			Input:      "How do you test a Go function that calls an external HTTP API and writes to a database? Show the interface-based mocking approach.",
			Expected:   []string{"interface", "mock", "httptest", "dependency injection", "test"},
			Tags:       []string{"testing"},
			Timeout:    30 * time.Second,
		},
	}
}
