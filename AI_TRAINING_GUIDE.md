# AI Training and Optimization Guide

## Overview

In this project, AI does not change system decisions. It only provides commentary and explanation. AI training can be done in two ways:

1. **Prompt Engineering** (Recommended - Fast and Easy)
2. **Fine-tuning** (Advanced - More Control)

---

## Training with Prompt Engineering

### Basic Logic

If you're using Ollama local LLM, you can train the AI with **prompt engineering** instead of fine-tuning. You provide reference to the AI by adding example data to the prompt in each chat request.

### Example Prompt Structure

```go
prompt := fmt.Sprintf(`You are a cybersecurity analyst. Consider the following examples:

1) Title: Tor forum leak
   Content: Database leaked
   Category: Data Leak
   Criticality: 8
   AI Comment: This is critical because user data was exposed.

2) Title: Ransomware attack
   Content: New ransomware variant detected
   Category: Malware Analysis
   Criticality: 9
   AI Comment: This is at a high criticality level because it's an active threat with spread potential.

3) Title: Security vulnerability disclosure
   Content: New CVE announced
   Category: Vulnerability Disclosure
   Criticality: 6
   AI Comment: Medium criticality level, known vulnerability with patches available.

Now comment on the incoming message in the same style (brief, clear, 2-3 sentences):

User Message: %s`, userMessage)
```

### Fetching Examples from Database

You can fetch examples from recent entries in the backend and add them to the prompt:

```go
// Example: Fetch examples from last 3 entries
examples := getRecentEntriesWithAI(3)
prompt := buildPromptWithExamples(userMessage, examples)
```

### Advantages

- Fast implementation
- No fine-tuning required
- Dynamic example updates
- Easy model changes

---

## Training with Fine-tuning (Advanced)

### Requirements

- Ollama Pro or fine-tuning supported model
- Example dataset (in JSON/CSV format)

### Dataset Preparation

**Format: JSON**
```json
[
  {
    "input": "Why is this entry critical?",
    "output": "This entry is critical because user data was exposed and there's an active threat."
  },
  {
    "input": "Why is the category Data Leak?",
    "output": "Category is Data Leak because a database leak was detected."
  }
]
```

**Format: CSV**
```csv
input,output
"Why is this entry critical?","This entry is critical because user data was exposed."
"Why is the category Data Leak?","Category is Data Leak because a database leak was detected."
```

### Fine-tuning Command (Ollama)

```bash
# Fine-tune the model
ollama create cti-analyst -f training_data.jsonl

# Use the fine-tuned model
# In docker-compose.yml:
AI_MODEL: cti-analyst
```

---

## Performance Optimization

### 1. Model Selection

**Fast Models (Recommended):**
- `mistral:7b-instruct` → `mistral:3b` (faster)
- `llama2:7b` → `llama2:3b` (faster)
- `phi` (Microsoft) - very fast, small

**Configuration:**
```yaml
environment:
  AI_MODEL: mistral:3b  # or phi
```

### 2. Token Limit

```go
req := OllamaGenerateRequest{
    Model:      s.model,
    Prompt:     prompt,
    Stream:     true,
    NumPredict: 150,  // 150-200 tokens (fast response)
}
```

### 3. Prompt Optimization

**Long Prompt:**
```
You are a Cyber Threat Intelligence (CTI) analyst. Your task:
1. Analyze and comment on cybersecurity threats
2. Provide explanations to users in technical and non-technical languages
... (very long)
```

**Short Prompt:**
```
You are a cybersecurity analyst. Make brief and clear comments (2-3 sentences max).
RULES: Only comment, don't make decisions. The system already made the decisions.
Message: {USER_MESSAGE}
```

### 4. Streaming Mode

Using streaming, the user doesn't wait in "thinking" mode, the answer comes gradually:

```javascript
// Streaming in frontend
const response = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ message, stream: true }),
});
```

### 5. Timeout Settings

```go
httpClient: &http.Client{
    Timeout: 30 * time.Second, // 5-10s ideal, 30s for safety
}
```

---

## 🔧 Local Performance Tips

### GPU Usage

Ollama works much faster with GPU support:

```bash
# Start Ollama with GPU
OLLAMA_GPU_ENABLED=true ollama serve
```

### RAM Optimization

- RAM requirements by model size:
  - 3B model: ~4GB RAM
  - 7B model: ~8GB RAM
  - 13B model: ~16GB RAM

### Request Throttling

If too many chat requests come in, add throttling:

```go
// Rate limiting in backend
var requestCount int
var lastReset time.Time

func checkRateLimit() bool {
    if time.Since(lastReset) > 1*time.Minute {
        requestCount = 0
        lastReset = time.Now()
    }
    if requestCount > 10 {
        return false // Rate limit exceeded
    }
    requestCount++
    return true
}
```

---

## Example Training Scenario

### Scenario 1: Training for Entry Comments

1. Fetch last 10 entries from database
2. Create example prompt for each entry
3. Add these examples to prompt in chat request
4. AI generates comments by referencing examples

### Scenario 2: Training for Chat

1. Identify frequently asked questions
2. Prepare example answers for each question
3. Add example Q&A to prompt
4. AI gives similar answers to similar questions

---

## Expected Results After Training

- AI comments are more targeted
- Prioritizes critical keywords
- Brief, clear, analyst-style answers
- Faster response times
- Consistent comment quality

---

## Important Notes

1. **AI Doesn't Make Decisions**: AI only provides commentary, doesn't determine category/criticality
2. **Non-blocking**: System continues if AI service is not working
3. **Soft Fallback**: User-friendly message shown in error cases
4. **Performance First**: Small model + token limit + streaming = fast UX

---

## Resources

- [Ollama Documentation](https://ollama.ai/docs)
- [Prompt Engineering Guide](https://www.promptingguide.ai/)
- [Fine-tuning Guide](https://ollama.ai/blog/fine-tuning)

