package benchmark

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const evaluationExcerptRunes = 1024

var answerColumnCandidates = []string{
	"reference_answer",
	"answer",
	"solution",
	"target",
	"label",
}

var promptColumnCandidates = []string{
	"prompt",
	"instruction",
	"question",
	"input",
	"context",
	"content",
	"conversation",
	"turn",
	"text",
}

var decimalNumberPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

type EvaluationInput struct {
	DatasetPath   string
	BenchmarkPath string
	AnswerColumn  string
	OutputDir     string
}

type EvaluationResult struct {
	Summary    api.LLMBenchmarkDatasetEvaluation
	ResultJSON string
	ResultCSV  string
	LogPath    string
}

type evaluationExcerpt struct {
	Text      string `json:"text"`
	RuneCount int    `json:"rune_count"`
	SHA256    string `json:"sha256"`
	Truncated bool   `json:"truncated"`
}

type answerValue struct {
	scorable bool
	key      string
	textHash string
	number   string
	excerpt  evaluationExcerpt
}

type datasetEntry struct {
	index    int
	answer   answerValue
	conflict bool
}

const (
	evaluationItemCorrect   = "correct"
	evaluationItemIncorrect = "incorrect"
	evaluationItemUnscored  = "unscored"
)

type evaluationItem struct {
	RequestID        string             `json:"request_id"`
	DatasetIndex     *int               `json:"dataset_index,omitempty"`
	State            string             `json:"state"`
	MatchType        string             `json:"match_type,omitempty"`
	ReferenceExcerpt *evaluationExcerpt `json:"reference_excerpt,omitempty"`
	PromptExcerpt    *evaluationExcerpt `json:"prompt_excerpt,omitempty"`
	OutputExcerpt    *evaluationExcerpt `json:"output_excerpt,omitempty"`
	Message          string             `json:"message,omitempty"`
}

var guideRequestArrayStates = map[string]string{
	"successful_requests": "successful",
	"successful":          "successful",
	"errored_requests":    "errored",
	"errored":             "errored",
	"incomplete_requests": "incomplete",
	"incomplete":          "incomplete",
	"timed_out_requests":  "timed_out",
	"timeout_requests":    "timed_out",
}

func normalizeWhitespace(value string) string {
	value = strings.TrimSpace(value)
	var out strings.Builder
	pendingSpace := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			pendingSpace = true
			continue
		}
		if pendingSpace && out.Len() > 0 {
			out.WriteByte(' ')
		}
		pendingSpace = false
		out.WriteRune(r)
	}
	return out.String()
}

func canonicalFold(value string) string {
	var out strings.Builder
	for _, r := range value {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		out.WriteRune(min)
	}
	return out.String()
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func parseExactNumber(value string) (string, bool) {
	if !decimalNumberPattern.MatchString(value) {
		return "", false
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return "", false
	}
	return number.RatString(), true
}

func makeEvaluationExcerpt(value string) evaluationExcerpt {
	runeCount := utf8.RuneCountInString(value)
	text := value
	truncated := runeCount > evaluationExcerptRunes
	if truncated {
		text = string([]rune(value)[:evaluationExcerptRunes])
	}
	return evaluationExcerpt{
		Text:      text,
		RuneCount: runeCount,
		SHA256:    sha256String(value),
		Truncated: truncated,
	}
}

func scalarText(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return "", false
	}
	switch value := value.(type) {
	case string:
		return value, true
	case json.Number:
		return value.String(), true
	case bool:
		return strconv.FormatBool(value), true
	default:
		return "", false
	}
}

func newAnswerValue(raw json.RawMessage) answerValue {
	text, ok := scalarText(raw)
	if !ok {
		return answerValue{key: "unscored"}
	}
	normalized := normalizeWhitespace(text)
	ret := answerValue{
		scorable: true,
		textHash: sha256String(canonicalFold(normalized)),
		excerpt:  makeEvaluationExcerpt(text),
	}
	if number, ok := parseExactNumber(normalized); ok {
		ret.number = number
		ret.key = "number:" + number
	} else {
		ret.key = "text:" + ret.textHash
	}
	return ret
}

func matchAnswer(reference answerValue, output string) (bool, string) {
	normalized := normalizeWhitespace(output)
	if reference.number != "" {
		if number, ok := parseExactNumber(normalized); ok {
			return number == reference.number, "number"
		}
	}
	return sha256String(canonicalFold(normalized)) == reference.textHash, "text"
}

func forEachDatasetRow(
	ctx context.Context,
	path string,
	visit func(index int, row map[string]json.RawMessage) error,
) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	for index := 0; ; index++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		row := map[string]json.RawMessage{}
		if err := decoder.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode dataset row %d: %w", index, err)
		}
		if err := visit(index, row); err != nil {
			return err
		}
	}
}

func detectAnswerColumn(ctx context.Context, path, explicit string) (string, string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		found := false
		err := forEachDatasetRow(ctx, path, func(_ int, row map[string]json.RawMessage) error {
			if _, ok := row[explicit]; ok {
				found = true
			}
			return nil
		})
		if err != nil {
			return "", "", err
		}
		if !found {
			return "", fmt.Sprintf("answer column %q not found", explicit), nil
		}
		return explicit, "", nil
	}

	found := map[string]bool{}
	err := forEachDatasetRow(ctx, path, func(_ int, row map[string]json.RawMessage) error {
		for _, candidate := range answerColumnCandidates {
			if _, ok := row[candidate]; ok {
				found[candidate] = true
			}
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	matches := make([]string, 0, len(found))
	for _, candidate := range answerColumnCandidates {
		if found[candidate] {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 0:
		return "", "no answer column found", nil
	case 1:
		return matches[0], "", nil
	default:
		return "", "multiple answer columns found: " + strings.Join(matches, ", "), nil
	}
}

func firstStringField(row map[string]json.RawMessage, candidates []string) (string, bool) {
	for _, candidate := range candidates {
		raw, ok := row[candidate]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return value, true
		}
	}
	return "", false
}

func promptDigest(prompt string) string {
	return sha256String(normalizeWhitespace(prompt))
}

func buildDatasetIndex(ctx context.Context, path, answerColumn string) (map[string]*datasetEntry, error) {
	// ponytail: the index is bounded by total_requests; move it to disk only if the 100k-request ceiling is proven too large.
	index := map[string]*datasetEntry{}
	err := forEachDatasetRow(ctx, path, func(rowIndex int, row map[string]json.RawMessage) error {
		prompt, ok := firstStringField(row, promptColumnCandidates)
		if !ok {
			return nil
		}
		answer := newAnswerValue(row[answerColumn])
		key := promptDigest(prompt)
		if existing := index[key]; existing != nil {
			if existing.answer.key != answer.key {
				existing.conflict = true
			}
			return nil
		}
		index[key] = &datasetEntry{
			index:  rowIndex,
			answer: answer,
		}
		return nil
	})
	return index, err
}

func walkGuideLLMRequests(
	ctx context.Context,
	path string,
	visit func(state string, raw json.RawMessage) error,
) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	return walkGuideLLMValue(ctx, decoder, "", visit)
}

func walkGuideLLMValue(
	ctx context.Context,
	decoder *json.Decoder,
	arrayState string,
	visit func(state string, raw json.RawMessage) error,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if arrayState != "" && delimiter != '[' {
		arrayState = ""
	}
	switch delimiter {
	case '{':
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("GuideLLM object key is not a string")
			}
			if err := walkGuideLLMValue(ctx, decoder, guideRequestArrayStates[key], visit); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		if arrayState != "" {
			for decoder.More() {
				var raw json.RawMessage
				if err := decoder.Decode(&raw); err != nil {
					return err
				}
				if err := visit(arrayState, raw); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		}
		for decoder.More() {
			if err := walkGuideLLMValue(ctx, decoder, "", visit); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected GuideLLM delimiter %q", delimiter)
	}
}

func decodeRawObject(raw json.RawMessage) map[string]json.RawMessage {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil {
		return object
	}
	var encoded string
	if json.Unmarshal(raw, &encoded) == nil && json.Unmarshal([]byte(encoded), &object) == nil {
		return object
	}
	return nil
}

func rawString(raw json.RawMessage) (string, bool) {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func extractPromptObject(object map[string]json.RawMessage) (string, bool) {
	if object == nil {
		return "", false
	}
	if prompt, ok := firstStringField(object, promptColumnCandidates); ok {
		return prompt, true
	}
	for _, key := range []string{"body", "payload", "request"} {
		if nested := decodeRawObject(object[key]); nested != nil {
			if prompt, ok := extractPromptObject(nested); ok {
				return prompt, true
			}
		}
	}
	var messages []map[string]json.RawMessage
	if json.Unmarshal(object["messages"], &messages) != nil {
		return "", false
	}
	userPrompt := ""
	userCount := 0
	for _, message := range messages {
		role, _ := rawString(message["role"])
		content, ok := rawString(message["content"])
		if role == "user" && ok {
			userPrompt = content
			userCount++
		}
	}
	return userPrompt, userCount == 1
}

func requestPrompt(object map[string]json.RawMessage) (string, bool) {
	for _, key := range []string{"request_args", "request", "input"} {
		if nested := decodeRawObject(object[key]); nested != nil {
			if prompt, ok := extractPromptObject(nested); ok {
				return prompt, true
			}
		}
	}
	return extractPromptObject(object)
}

func extractOutputRaw(raw json.RawMessage) (string, bool) {
	object := decodeRawObject(raw)
	if object == nil {
		return rawString(raw)
	}
	for _, key := range []string{"content", "text", "output", "generated_text"} {
		if value, ok := rawString(object[key]); ok {
			return value, true
		}
	}
	for _, key := range []string{"message", "body", "response"} {
		if value, ok := extractOutputRaw(object[key]); ok {
			return value, true
		}
	}
	var choices []map[string]json.RawMessage
	if json.Unmarshal(object["choices"], &choices) == nil {
		for _, choice := range choices {
			if value, ok := extractOutputRaw(choice["message"]); ok {
				return value, true
			}
			if value, ok := rawString(choice["text"]); ok {
				return value, true
			}
		}
	}
	return "", false
}

func requestOutput(object map[string]json.RawMessage) (string, bool) {
	for _, key := range []string{"response", "response_args", "output", "result"} {
		if value, ok := extractOutputRaw(object[key]); ok {
			return value, true
		}
	}
	return "", false
}

func requestID(object map[string]json.RawMessage, ordinal int) string {
	for _, key := range []string{"id", "request_id", "request_uuid"} {
		if value, ok := rawString(object[key]); ok && value != "" {
			return value
		}
	}
	return fmt.Sprintf("request-%d", ordinal)
}

func intPointer(value int) *int {
	return &value
}

func scoreGuideRequest(
	state string,
	raw json.RawMessage,
	index map[string]*datasetEntry,
	ordinal int,
) evaluationItem {
	object := decodeRawObject(raw)
	item := evaluationItem{RequestID: requestID(object, ordinal)}
	prompt, ok := requestPrompt(object)
	if !ok {
		item.State = evaluationItemUnscored
		item.Message = "single-turn prompt not found in request_args"
		return item
	}
	promptExcerpt := makeEvaluationExcerpt(prompt)
	item.PromptExcerpt = &promptExcerpt
	entry := index[promptDigest(prompt)]
	if entry == nil {
		item.State = evaluationItemUnscored
		item.Message = "prompt not found in evaluation dataset"
		return item
	}
	item.DatasetIndex = intPointer(entry.index)
	item.ReferenceExcerpt = &entry.answer.excerpt
	if entry.conflict {
		item.State = evaluationItemUnscored
		item.Message = "duplicate prompt has conflicting reference answers"
		return item
	}
	if !entry.answer.scorable {
		item.State = evaluationItemUnscored
		item.ReferenceExcerpt = nil
		item.Message = "reference answer is missing or not a scalar"
		return item
	}
	if state != "successful" {
		item.State = evaluationItemIncorrect
		item.Message = "request " + state
		return item
	}
	output, ok := requestOutput(object)
	if !ok || normalizeWhitespace(output) == "" {
		item.State = evaluationItemIncorrect
		item.Message = "successful request has empty output"
		return item
	}
	outputExcerpt := makeEvaluationExcerpt(output)
	item.OutputExcerpt = &outputExcerpt
	matched, matchType := matchAnswer(entry.answer, output)
	item.MatchType = matchType
	if matched {
		item.State = evaluationItemCorrect
	} else {
		item.State = evaluationItemIncorrect
	}
	return item
}

func addEvaluationItem(summary *api.LLMBenchmarkDatasetEvaluation, item evaluationItem) {
	summary.RequestTotal++
	switch item.State {
	case evaluationItemCorrect:
		summary.Correct++
	case evaluationItemIncorrect:
		summary.Incorrect++
	case evaluationItemUnscored:
		summary.Unscored++
	}
}

var evaluationCSVHeader = []string{
	"request_id", "dataset_index", "state", "match_type", "message",
	"prompt_excerpt", "prompt_rune_count", "prompt_sha256", "prompt_truncated",
	"reference_excerpt", "reference_rune_count", "reference_sha256", "reference_truncated",
	"output_excerpt", "output_rune_count", "output_sha256", "output_truncated",
}

func excerptCSV(value *evaluationExcerpt) []string {
	if value == nil {
		return []string{"", "", "", ""}
	}
	return []string{
		value.Text,
		strconv.Itoa(value.RuneCount),
		value.SHA256,
		strconv.FormatBool(value.Truncated),
	}
}

func evaluationCSVRow(item evaluationItem) []string {
	datasetIndex := ""
	if item.DatasetIndex != nil {
		datasetIndex = strconv.Itoa(*item.DatasetIndex)
	}
	row := []string{item.RequestID, datasetIndex, item.State, item.MatchType, item.Message}
	row = append(row, excerptCSV(item.PromptExcerpt)...)
	row = append(row, excerptCSV(item.ReferenceExcerpt)...)
	row = append(row, excerptCSV(item.OutputExcerpt)...)
	return row
}

func EvaluateDataset(ctx context.Context, input EvaluationInput) (ret EvaluationResult, retErr error) {
	if err := os.MkdirAll(input.OutputDir, 0755); err != nil {
		return ret, err
	}
	ret.LogPath = filepath.Join(input.OutputDir, "evaluation.log")
	logFile, err := os.Create(ret.LogPath)
	if err != nil {
		return ret, err
	}
	defer logFile.Close()
	defer func() {
		if retErr == nil {
			return
		}
		ret.Summary.State = api.LLMBenchmarkEvaluationStateError
		ret.Summary.Message = retErr.Error()
		_, _ = fmt.Fprintf(logFile, "evaluation error: %s\n", retErr)
	}()

	column, skipMessage, err := detectAnswerColumn(ctx, input.DatasetPath, input.AnswerColumn)
	if err != nil {
		return ret, err
	}
	if skipMessage != "" {
		ret.Summary.State = api.LLMBenchmarkEvaluationStateSkipped
		ret.Summary.Message = skipMessage
		_, _ = fmt.Fprintf(logFile, "evaluation skipped: %s\n", skipMessage)
		return ret, nil
	}
	ret.Summary.AnswerColumn = column
	_, _ = fmt.Fprintf(logFile, "answer column: %s\n", column)

	index, err := buildDatasetIndex(ctx, input.DatasetPath, column)
	if err != nil {
		return ret, err
	}

	jsonPath := filepath.Join(input.OutputDir, "evaluation.json")
	csvPath := filepath.Join(input.OutputDir, "evaluation.csv")
	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		return ret, err
	}
	csvFile, err := os.Create(csvPath)
	if err != nil {
		_ = jsonFile.Close()
		_ = os.Remove(jsonPath)
		return ret, err
	}
	reportsComplete := false
	defer func() {
		_ = jsonFile.Close()
		_ = csvFile.Close()
		if !reportsComplete {
			_ = os.Remove(jsonPath)
			_ = os.Remove(csvPath)
		}
	}()

	if _, err := io.WriteString(jsonFile, "{\"items\":["); err != nil {
		return ret, err
	}
	jsonEncoder := json.NewEncoder(jsonFile)
	csvWriter := csv.NewWriter(csvFile)
	if err := csvWriter.Write(evaluationCSVHeader); err != nil {
		return ret, err
	}
	first := true
	ordinal := 0
	err = walkGuideLLMRequests(ctx, input.BenchmarkPath, func(state string, raw json.RawMessage) error {
		ordinal++
		item := scoreGuideRequest(state, raw, index, ordinal)
		addEvaluationItem(&ret.Summary, item)
		if !first {
			if _, err := io.WriteString(jsonFile, ","); err != nil {
				return err
			}
		}
		first = false
		if err := jsonEncoder.Encode(item); err != nil {
			return err
		}
		return csvWriter.Write(evaluationCSVRow(item))
	})
	if err != nil {
		return ret, err
	}
	if ret.Summary.RequestTotal == 0 {
		err = errors.New("no GuideLLM request records found")
		return ret, err
	}

	ret.Summary.Evaluated = ret.Summary.Correct + ret.Summary.Incorrect
	if ret.Summary.Evaluated > 0 {
		ret.Summary.Accuracy = float64(ret.Summary.Correct) / float64(ret.Summary.Evaluated)
	} else {
		ret.Summary.Message = "no scorable requests"
	}
	ret.Summary.State = api.LLMBenchmarkEvaluationStateCompleted
	if _, err := io.WriteString(jsonFile, "],\"summary\":"); err != nil {
		return ret, err
	}
	if err := jsonEncoder.Encode(ret.Summary); err != nil {
		return ret, err
	}
	if _, err := io.WriteString(jsonFile, "}"); err != nil {
		return ret, err
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return ret, err
	}
	if err := jsonFile.Close(); err != nil {
		return ret, err
	}
	if err := csvFile.Close(); err != nil {
		return ret, err
	}
	reportsComplete = true
	ret.ResultJSON = jsonPath
	ret.ResultCSV = csvPath
	_, _ = fmt.Fprintf(
		logFile,
		"requests=%d evaluated=%d correct=%d incorrect=%d unscored=%d accuracy=%g\n",
		ret.Summary.RequestTotal,
		ret.Summary.Evaluated,
		ret.Summary.Correct,
		ret.Summary.Incorrect,
		ret.Summary.Unscored,
		ret.Summary.Accuracy,
	)
	return ret, nil
}
