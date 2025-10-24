package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const version = "1.0.0"

// ForgejoWebhook represents the incoming webhook from Forgejo/Gitea
type ForgejoWebhook struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"head_commit"`
	Pusher struct {
		Username string `json:"username"`
	} `json:"pusher"`
}

// BuildkitePayload represents the payload to send to Buildkite
type BuildkitePayload struct {
	Commit  string            `json:"commit"`
	Branch  string            `json:"branch"`
	Message string            `json:"message"`
	Author  *BuildkiteAuthor  `json:"author,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// BuildkiteAuthor represents the commit author
type BuildkiteAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var (
	buildkiteOrg   string
	buildkiteToken string
	port           string
	logVerbose     bool
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	// Initialize configuration from environment variables
	buildkiteOrg = os.Getenv("BUILDKITE_ORG")
	buildkiteToken = os.Getenv("BUILDKITE_TOKEN")
	port = getEnv("WEBHOOK_PORT", "8080")
	logVerbose = getEnv("LOG_VERBOSE", "false") == "true"

	if buildkiteOrg == "" || buildkiteToken == "" {
		log.Fatal("Error: BUILDKITE_ORG and BUILDKITE_TOKEN environment variables must be set\n" +
			"Get token from: https://buildkite.com/user/api-access-tokens (requires write_builds scope)")
	}

	http.HandleFunc("/webhook/", webhookHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rootHandler)

	log.Printf("🚀 Buildkite-Forgejo Webhook Bridge v%s", version)
	log.Printf("📡 Listening on port %s", port)
	log.Printf("🏢 Buildkite organization: %s", buildkiteOrg)
	log.Printf("📝 Verbose logging: %v", logVerbose)
	log.Println()
	log.Printf("Configure Forgejo webhook to: http://your-host:%s/webhook/<pipeline-slug>", port)
	log.Printf("Example: http://your-host:%s/webhook/my-app", port)
	log.Println()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Buildkite-Forgejo Webhook Bridge</title>
    <style>
        body { font-family: system-ui; max-width: 800px; margin: 50px auto; padding: 20px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        h1 { color: #333; }
        .status { color: #28a745; }
    </style>
</head>
<body>
    <h1>🚀 Buildkite-Forgejo Webhook Bridge</h1>
    <p class="status">✓ Service is running (v%s)</p>
    <h2>Configuration</h2>
    <ul>
        <li><strong>Buildkite Org:</strong> %s</li>
        <li><strong>Webhook URL:</strong> <code>http://your-host:%s/webhook/&lt;pipeline-slug&gt;</code></li>
    </ul>
    <h2>Setup Instructions</h2>
    <ol>
        <li>Go to your Forgejo repository settings</li>
        <li>Navigate to Webhooks → Add Webhook</li>
        <li>Set URL to: <code>http://your-host:%s/webhook/&lt;pipeline-slug&gt;</code></li>
        <li>Set Content Type: <code>application/json</code></li>
        <li>Select trigger: <strong>Push events</strong></li>
        <li>Save and test!</li>
    </ol>
    <h2>Endpoints</h2>
    <ul>
        <li><code>/</code> - This page</li>
        <li><code>/health</code> - Health check endpoint</li>
        <li><code>/webhook/&lt;pipeline-slug&gt;</code> - Webhook receiver</li>
    </ul>
</body>
</html>`, version, buildkiteOrg, port, port)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": version,
		"org":     buildkiteOrg,
	})
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pipeline slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/webhook/")
	if path == "" || path == "webhook/" {
		http.Error(w, "Pipeline slug required in URL: /webhook/<pipeline-slug>", http.StatusBadRequest)
		return
	}

	// Parse Forgejo webhook payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if logVerbose {
		log.Printf("📦 Received payload: %s", string(body))
	}

	var webhook ForgejoWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.Printf("❌ Error parsing webhook: %v", err)
		http.Error(w, "Bad request: invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract branch from ref (refs/heads/main -> main)
	branch := strings.TrimPrefix(webhook.Ref, "refs/heads/")
	commitShort := webhook.HeadCommit.ID
	if len(commitShort) > 7 {
		commitShort = commitShort[:7]
	}

	log.Printf("📨 Webhook: repo=%s, branch=%s, commit=%s, author=%s",
		webhook.Repository.FullName, branch, commitShort, webhook.Pusher.Username)

	// Trigger Buildkite build
	if err := triggerBuild(path, branch, webhook.HeadCommit.ID, webhook.HeadCommit.Message, &webhook); err != nil {
		log.Printf("❌ Failed to trigger build: %v", err)
		http.Error(w, fmt.Sprintf("Failed to trigger build: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "Build triggered successfully",
		"pipeline": path,
		"branch":   branch,
		"commit":   commitShort,
	})
}

func triggerBuild(pipeline, branch, commit, message string, webhook *ForgejoWebhook) error {
	url := fmt.Sprintf("https://api.buildkite.com/v2/organizations/%s/pipelines/%s/builds", buildkiteOrg, pipeline)
	
	if logVerbose {
		log.Printf("🔍 Debug: buildkiteOrg='%s', pipeline='%s'", buildkiteOrg, pipeline)
		log.Printf("🔍 Debug: URL='%s'", url)
	}

	payload := BuildkitePayload{
		Commit:  commit,
		Branch:  branch,
		Message: message,
		Author: &BuildkiteAuthor{
			Name:  webhook.HeadCommit.Author.Name,
			Email: webhook.HeadCommit.Author.Email,
		},
		Env: map[string]string{
			"FORGEJO_PUSHER":   webhook.Pusher.Username,
			"FORGEJO_REPO":     webhook.Repository.FullName,
			"FORGEJO_REPO_NAME": webhook.Repository.Name,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if logVerbose {
		log.Printf("📤 Buildkite payload: %s", string(jsonData))
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+buildkiteToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("buildkite API returned %d: %s", resp.StatusCode, string(respBody))
	}

	if logVerbose {
		log.Printf("📥 Buildkite response: %s", string(respBody))
	}

	log.Printf("✅ Build triggered: %s/%s (branch: %s, commit: %s)", buildkiteOrg, pipeline, branch, commit[:7])
	return nil
}
