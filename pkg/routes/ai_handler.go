// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
// AI handler: Voice → STT (Groq Whisper) → Tasks (Gemini 2.0 Flash).
// Text/Image → Tasks (Gemini 2.0 Flash).
// BOQ generation, Phase suggestions, Usage tracking, Freemium enforcement.
package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"code.vikunja.io/api/pkg/ai"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/labstack/echo/v5"
	"xorm.io/xorm"
)

// ─── Startup Validation ────────────────────────────────────────────────────

// ValidateArchTaskConfig checks for required environment variables.
// Logs warnings if keys are missing; does NOT prevent startup.
func ValidateArchTaskConfig() {
	if os.Getenv("GROQ_API_KEY") == "" {
		log.Warning("[ArchTask] GROQ_API_KEY is not set. Voice transcription will be unavailable.")
	}
	if os.Getenv("GEMINI_API_KEY") == "" {
		log.Warning("[ArchTask] GEMINI_API_KEY is not set. AI task generation will be unavailable.")
	}
}

// ─── Route Registration ───────────────────────────────────────────────────

// RegisterAIRoutes registers all ArchTask AI endpoints on the given authenticated Echo group.
func RegisterAIRoutes(g *echo.Group) {
	// Input → Tasks
	g.POST("/input/voice", handleAIVoice)
	g.POST("/input/text", handleAIText)
	g.POST("/input/image", handleAIImage)

	// BOQ generation
	g.POST("/projects/:id/generate-boq", handleGenerateBOQ)

	// AI Task Suggestions
	g.GET("/projects/:id/suggest-tasks", handleSuggestTasks)

	// Project Templates
	g.POST("/projects/:id/apply-template", handleApplyTemplate)
	g.GET("/projects/:id/phase-summary", handlePhaseSummary)
	g.GET("/project-types", handleGetProjectTypes)

	// Usage Stats
	g.GET("/usage", handleGetUsage)

	log.Info("[ArchTask] AI routes registered under /api/v1/archtask")
}

// ─── Internal Types ───────────────────────────────────────────────────────

type archErrResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type aiTask struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ArchPhase   string   `json:"arch_phase"`
	DueDate     *string  `json:"due_date"`
	Priority    int64    `json:"priority"`
	Labels      []string `json:"labels"`
}

type geminiPart map[string]interface{}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction geminiContent   `json:"systemInstruction"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────

func archErr(c *echo.Context, status int, msg, code string) error {
	return c.JSON(status, archErrResponse{Error: msg, Code: code})
}

func requireGeminiKey(c *echo.Context) (string, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return "", archErr(c, http.StatusServiceUnavailable,
			"AI service not configured. Please contact the administrator.",
			"ERR_GEMINI_KEY_MISSING")
	}
	return key, nil
}

func requireGroqKey(c *echo.Context) (string, error) {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return "", archErr(c, http.StatusServiceUnavailable,
			"Voice transcription not configured. Please contact the administrator.",
			"ERR_GROQ_KEY_MISSING")
	}
	return key, nil
}

func callGeminiRaw(systemPrompt, userText string, extraParts []geminiPart, apiKey string) (string, error) {
	parts := []geminiPart{{"text": userText}}
	parts = append(parts, extraParts...)

	payload := geminiRequest{
		SystemInstruction: geminiContent{Parts: []geminiPart{{"text": systemPrompt}}},
		Contents:          []geminiContent{{Parts: parts}},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini http call: %w", err)
	}
	defer resp.Body.Close()

	var geminiRes map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&geminiRes); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}

	candidates, ok := geminiRes["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		raw, _ := json.Marshal(geminiRes)
		return "", fmt.Errorf("unexpected gemini response: %s", string(raw))
	}

	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing content in gemini response")
	}

	parts2, ok := content["parts"].([]interface{})
	if !ok || len(parts2) == 0 {
		return "", fmt.Errorf("missing parts in gemini content")
	}

	textResult, ok := parts2[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("missing text in gemini parts")
	}

	// Strip markdown code fences if present
	textResult = strings.TrimSpace(textResult)
	if strings.HasPrefix(textResult, "```json") {
		textResult = strings.TrimPrefix(textResult, "```json")
		textResult = strings.TrimSuffix(strings.TrimSpace(textResult), "```")
	} else if strings.HasPrefix(textResult, "```") {
		textResult = strings.TrimPrefix(textResult, "```")
		textResult = strings.TrimSuffix(strings.TrimSpace(textResult), "```")
	}

	return strings.TrimSpace(textResult), nil
}

// parseTasks parses Gemini JSON response into a slice of aiTask.
func parseTasks(jsonText string) ([]aiTask, error) {
	var result struct {
		Tasks []aiTask `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return nil, fmt.Errorf("parse tasks json: %w", err)
	}
	return result.Tasks, nil
}

// ─── Handlers ────────────────────────────────────────────────────────────

// handleAIVoice transcribes audio via Groq Whisper then generates tasks via Gemini.
func handleAIVoice(c *echo.Context) error {
	groqKey, err := requireGroqKey(c)
	if err != nil {
		return err
	}
	geminiKey, err := requireGeminiKey(c)
	if err != nil {
		return err
	}

	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	s := db.NewSession()
	defer s.Close()

	canUse, remaining, err := models.CanUserUseAI(s, currentUser.ID)
	if err != nil {
		log.Errorf("[ArchTask] Usage check error for user %d: %v", currentUser.ID, err)
		return archErr(c, http.StatusInternalServerError, "Failed to check usage limit", "ERR_USAGE_CHECK")
	}
	if !canUse {
		return archErr(c, http.StatusPaymentRequired,
			fmt.Sprintf("لقد استنفدت حد الاستخدام المجاني (%d عملية/شهر). / You have reached the free usage limit (%d ops/month).", models.FreeMonthlyLimit, models.FreeMonthlyLimit),
			"ERR_USAGE_LIMIT_REACHED")
	}

	file, err := c.FormFile("audio")
	if err != nil {
		return archErr(c, http.StatusBadRequest, "Audio file is required", "ERR_NO_AUDIO")
	}

	projectIDStr := c.FormValue("project_id")
	projectID, parseErr := strconv.ParseInt(projectIDStr, 10, 64)
	if parseErr != nil || projectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Valid project_id is required", "ERR_INVALID_PROJECT_ID")
	}

	src, err := file.Open()
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to read audio file", "ERR_FILE_READ")
	}
	defer src.Close()

	// Build multipart for Groq Whisper
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", file.Filename+".webm")
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to prepare audio upload", "ERR_MULTIPART")
	}
	if _, err = io.Copy(part, src); err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to stream audio", "ERR_STREAM")
	}
	_ = writer.WriteField("model", "whisper-large-v3-turbo")
	_ = writer.WriteField("response_format", "json")
	writer.Close()

	groqReq, _ := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/audio/transcriptions", &body)
	groqReq.Header.Set("Authorization", "Bearer "+groqKey)
	groqReq.Header.Set("Content-Type", writer.FormDataContentType())

	groqResp, err := http.DefaultClient.Do(groqReq)
	if err != nil {
		log.Errorf("[ArchTask] Groq STT network error: %v", err)
		return archErr(c, http.StatusServiceUnavailable, "Voice transcription service unavailable / خدمة تحويل الصوت غير متاحة", "ERR_GROQ_STT")
	}
	if groqResp.StatusCode != http.StatusOK {
		log.Errorf("[ArchTask] Groq STT HTTP %d error", groqResp.StatusCode)
		return archErr(c, http.StatusServiceUnavailable, "Voice transcription service unavailable / خدمة تحويل الصوت غير متاحة", "ERR_GROQ_STT")
	}
	defer groqResp.Body.Close()

	var groqResult map[string]interface{}
	if err = json.NewDecoder(groqResp.Body).Decode(&groqResult); err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to parse transcription", "ERR_GROQ_PARSE")
	}

	transcript, ok := groqResult["text"].(string)
	if !ok || transcript == "" {
		return archErr(c, http.StatusUnprocessableEntity, "Could not transcribe audio", "ERR_EMPTY_TRANSCRIPT")
	}

	log.Infof("[ArchTask] Voice transcribed for user %d: %q", currentUser.ID, transcript[:min(50, len(transcript))])

	return processAndSaveTasks(c, s, currentUser.ID, projectID, transcript, nil, geminiKey, "voice", remaining)
}

// handleAIText generates tasks from a text prompt.
func handleAIText(c *echo.Context) error {
	geminiKey, err := requireGeminiKey(c)
	if err != nil {
		return err
	}

	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	var reqBody struct {
		Text      string `json:"text"`
		ProjectID int64  `json:"project_id"`
	}
	if err = c.Bind(&reqBody); err != nil || reqBody.Text == "" || reqBody.ProjectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Fields 'text' and 'project_id' are required", "ERR_INVALID_BODY")
	}

	s := db.NewSession()
	defer s.Close()

	canUse, remaining, err := models.CanUserUseAI(s, currentUser.ID)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to check usage limit", "ERR_USAGE_CHECK")
	}
	if !canUse {
		return archErr(c, http.StatusPaymentRequired,
			fmt.Sprintf("لقد استنفدت حد الاستخدام المجاني (%d عملية/شهر).", models.FreeMonthlyLimit),
			"ERR_USAGE_LIMIT_REACHED")
	}

	return processAndSaveTasks(c, s, currentUser.ID, reqBody.ProjectID, reqBody.Text, nil, geminiKey, "text", remaining)
}

// handleAIImage generates tasks from an image (base64) description.
func handleAIImage(c *echo.Context) error {
	geminiKey, err := requireGeminiKey(c)
	if err != nil {
		return err
	}

	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	var reqBody struct {
		Image     string `json:"image"`      // base64 encoded
		MimeType  string `json:"mimeType"`   // e.g. "image/jpeg"
		ProjectID int64  `json:"project_id"`
	}
	if err = c.Bind(&reqBody); err != nil || reqBody.Image == "" || reqBody.ProjectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Fields 'image', 'mimeType', and 'project_id' are required", "ERR_INVALID_BODY")
	}

	s := db.NewSession()
	defer s.Close()

	canUse, remaining, err := models.CanUserUseAI(s, currentUser.ID)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to check usage limit", "ERR_USAGE_CHECK")
	}
	if !canUse {
		return archErr(c, http.StatusPaymentRequired,
			fmt.Sprintf("لقد استنفدت حد الاستخدام المجاني (%d عملية/شهر).", models.FreeMonthlyLimit),
			"ERR_USAGE_LIMIT_REACHED")
	}

	imagePart := geminiPart{
		"inlineData": map[string]interface{}{
			"mimeType": reqBody.MimeType,
			"data":     reqBody.Image,
		},
	}

	return processAndSaveTasks(c, s, currentUser.ID, reqBody.ProjectID,
		"استخرج المهام المعمارية من هذه الصورة. / Extract architectural tasks from this image.",
		[]geminiPart{imagePart}, geminiKey, "image", remaining)
}

// processAndSaveTasks is the shared pipeline: Gemini call → parse → save to DB → record usage.
func processAndSaveTasks(c *echo.Context, s *xorm.Session, userID, projectID int64, prompt string, extraParts []geminiPart, geminiKey, opType string, remaining int) error {
	jsonText, err := callGeminiRaw(ai.ArchSystemPrompt, prompt, extraParts, geminiKey)
	if err != nil {
		log.Errorf("[ArchTask] Gemini error for user %d: %v", userID, err)
		return archErr(c, http.StatusServiceUnavailable, "AI task generation failed. Please try again. / فشلت خدمة الذكاء الاصطناعي. حاول مجدداً.", "ERR_GEMINI_CALL")
	}

	tasks, err := parseTasks(jsonText)
	if err != nil {
		log.Errorf("[ArchTask] Parse error: %v | raw: %s", err, jsonText[:min(200, len(jsonText))])
		return archErr(c, http.StatusInternalServerError, "Failed to parse AI response / فشل في تحليل استجابة الذكاء الاصطناعي", "ERR_PARSE")
	}

	if len(tasks) == 0 {
		return archErr(c, http.StatusUnprocessableEntity, "No tasks were generated from your input / لم يتم توليد أي مهام من المدخل", "ERR_NO_TASKS")
	}

	// Load the auth user for task creation permission checks
	createdByUser, err := user.GetUserByID(s, userID)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to load user", "ERR_USER_LOAD")
	}

	if err = s.Begin(); err != nil {
		return archErr(c, http.StatusInternalServerError, "Database error", "ERR_DB")
	}

	var savedTasks []*models.Task
	for _, t := range tasks {
		task := &models.Task{
			Title:       t.Title,
			Description: t.Description,
			ProjectID:   projectID,
			Priority:    t.Priority,
			ArchPhase:   t.ArchPhase,
			AIGenerated: true,
		}
		if t.Priority < 1 {
			task.Priority = 2
		}
		if t.Priority > 5 {
			task.Priority = 5
		}

		if err = task.Create(s, createdByUser); err != nil {
			log.Errorf("[ArchTask] Failed to save task '%s': %v", task.Title, err)
			continue
		}
		savedTasks = append(savedTasks, task)
	}

	if err = s.Commit(); err != nil {
		_ = s.Rollback()
		return archErr(c, http.StatusInternalServerError, "Failed to save tasks", "ERR_DB_COMMIT")
	}

	// Record usage (non-fatal if it fails)
	if recErr := models.RecordAIUsage(s, userID, opType, 0); recErr != nil {
		log.Warningf("[ArchTask] Failed to record usage for user %d: %v", userID, recErr)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"created":           len(savedTasks),
		"tasks":             savedTasks,
		"remaining_free_ops": remaining - 1,
	})
}

// handleGenerateBOQ generates a Bill of Quantities from all project tasks using Gemini.
func handleGenerateBOQ(c *echo.Context) error {
	geminiKey, err := requireGeminiKey(c)
	if err != nil {
		return err
	}

	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil || projectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Invalid project ID", "ERR_INVALID_PROJECT_ID")
	}

	s := db.NewSession()
	defer s.Close()

	canUse, _, err := models.CanUserUseAI(s, currentUser.ID)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to check usage limit", "ERR_USAGE_CHECK")
	}
	if !canUse {
		return archErr(c, http.StatusPaymentRequired,
			fmt.Sprintf("لقد استنفدت حد الاستخدام المجاني (%d عملية/شهر).", models.FreeMonthlyLimit),
			"ERR_USAGE_LIMIT_REACHED")
	}

	// Fetch all tasks for the project
	var tasks []*models.Task
	err = s.Where("project_id = ?", projectID).Find(&tasks)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to fetch project tasks", "ERR_DB")
	}

	if len(tasks) == 0 {
		return archErr(c, http.StatusUnprocessableEntity, "No tasks found in this project / لا توجد مهام في هذا المشروع", "ERR_NO_TASKS")
	}

	// Build task list for prompt
	var taskLines []string
	for _, t := range tasks {
		line := fmt.Sprintf("[%s] %s: %s", t.ArchPhase, t.Title, t.Description)
		taskLines = append(taskLines, line)
	}
	taskListText := strings.Join(taskLines, "\n")

	jsonText, err := callGeminiRaw(ai.BOQSystemPrompt, taskListText, nil, geminiKey)
	if err != nil {
		log.Errorf("[ArchTask] BOQ Gemini error: %v", err)
		return archErr(c, http.StatusServiceUnavailable, "BOQ generation failed / فشل توليد جدول الكميات", "ERR_GEMINI_BOQ")
	}

	// Record usage
	if recErr := models.RecordAIUsage(s, currentUser.ID, "boq", 0); recErr != nil {
		log.Warningf("[ArchTask] Failed to record BOQ usage: %v", recErr)
	}

	// Return raw BOQ JSON
	return c.String(http.StatusOK, jsonText)
}

// handleSuggestTasks returns AI-suggested standard tasks for a project phase.
func handleSuggestTasks(c *echo.Context) error {
	geminiKey, err := requireGeminiKey(c)
	if err != nil {
		return err
	}

	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil || projectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Invalid project ID", "ERR_INVALID_PROJECT_ID")
	}

	phase := c.QueryParam("phase")
	if phase == "" {
		phase = "SD"
	}
	projectType := c.QueryParam("project_type")
	if projectType == "" {
		projectType = "default"
	}

	s := db.NewSession()
	defer s.Close()

	canUse, remaining, err := models.CanUserUseAI(s, currentUser.ID)
	if err != nil {
		return archErr(c, http.StatusInternalServerError, "Failed to check usage limit", "ERR_USAGE_CHECK")
	}
	if !canUse {
		return archErr(c, http.StatusPaymentRequired,
			fmt.Sprintf("لقد استنفدت حد الاستخدام المجاني (%d عملية/شهر).", models.FreeMonthlyLimit),
			"ERR_USAGE_LIMIT_REACHED")
	}

	prompt := fmt.Sprintf(
		"نوع المشروع / Project type: %s\nالمرحلة / Phase: %s\n%s",
		projectType, phase,
		ai.GetProjectTypeSuggestionsContext(projectType, phase),
	)

	jsonText, err := callGeminiRaw(ai.SuggestionsSystemPrompt, prompt, nil, geminiKey)
	if err != nil {
		log.Errorf("[ArchTask] Suggestions Gemini error: %v", err)
		return archErr(c, http.StatusServiceUnavailable, "Suggestions generation failed", "ERR_GEMINI_SUGGEST")
	}

	if recErr := models.RecordAIUsage(s, currentUser.ID, "suggest", 0); recErr != nil {
		log.Warningf("[ArchTask] Failed to record suggest usage: %v", recErr)
	}

	// Return raw JSON + remaining ops
	return c.JSON(http.StatusOK, map[string]interface{}{
		"project_id":         projectID,
		"phase":              phase,
		"project_type":       projectType,
		"suggestions_json":   jsonText,
		"remaining_free_ops": remaining - 1,
	})
}

// handleApplyTemplate applies an architectural project template to a project.
func handleApplyTemplate(c *echo.Context) error {
	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil || projectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Invalid project ID", "ERR_INVALID_PROJECT_ID")
	}

	var reqBody struct {
		ProjectType string `json:"project_type"`
	}
	if err = c.Bind(&reqBody); err != nil || reqBody.ProjectType == "" {
		return archErr(c, http.StatusBadRequest, "Field 'project_type' is required", "ERR_INVALID_BODY")
	}

	created, err := models.ApplyArchitecturalTemplate(projectID, reqBody.ProjectType, currentUser)
	if err != nil {
		log.Errorf("[ArchTask] Template apply error for project %d: %v", projectID, err)
		return archErr(c, http.StatusInternalServerError, "Failed to apply template / فشل تطبيق القالب", "ERR_TEMPLATE")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"created":      created,
		"project_type": reqBody.ProjectType,
		"message":      fmt.Sprintf("تم إنشاء %d مهمة من القالب بنجاح / %d template tasks created successfully", created, created),
	})
}

// handlePhaseSummary returns task counts per arch phase for a project.
func handlePhaseSummary(c *echo.Context) error {
	_, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil || projectID <= 0 {
		return archErr(c, http.StatusBadRequest, "Invalid project ID", "ERR_INVALID_PROJECT_ID")
	}

	s := db.NewSession()
	defer s.Close()

	summary, err := models.GetProjectPhaseSummary(s, projectID)
	if err != nil {
		log.Errorf("[ArchTask] Phase summary error for project %d: %v", projectID, err)
		return archErr(c, http.StatusInternalServerError, "Failed to get phase summary", "ERR_PHASE_SUMMARY")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"phases":     summary,
	})
}

// handleGetProjectTypes returns the list of supported architectural project types.
func handleGetProjectTypes(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"project_types": models.GetSupportedProjectTypes(),
	})
}

// handleGetUsage returns the current user's AI usage stats.
func handleGetUsage(c *echo.Context) error {
	currentUser, err := user.GetCurrentUser(c)
	if err != nil {
		return archErr(c, http.StatusUnauthorized, "Authentication required", "ERR_AUTH")
	}

	stats, err := models.GetUsageStats(currentUser.ID)
	if err != nil {
		log.Errorf("[ArchTask] Usage stats error for user %d: %v", currentUser.ID, err)
		return archErr(c, http.StatusInternalServerError, "Failed to get usage stats", "ERR_USAGE_STATS")
	}

	return c.JSON(http.StatusOK, stats)
}

// min helper removed — Go 1.21+ provides min() as a builtin.

