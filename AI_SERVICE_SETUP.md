# AI Service Setup Guide

## Overview

The AI component in this project is **supportive only** - it provides commentary and explanation, but **never makes decisions**. The system determines categories and criticality scores independently.

## AI Service Requirements

### Endpoint Specification

Your AI service should implement the following endpoint:

**Endpoint:** `POST /analyze`

**Request Format:**
```json
{
  "title": "Entry title",
  "content": "Entry content",
  "category": "Category name",
  "criticality_score": 85
}
```

**Response Format:**
```json
{
  "analysis": "AI commentary text here"
}
```

### Important Constraints

The AI service **SHOULD NOT**:
- Determine categories
- Calculate criticality scores
- Make system decisions
- Suggest different categories or scores

The AI service **SHOULD**:
- Comment on why the system assigned this category
- Explain why the criticality score is what it is
- Provide context for non-technical decision makers
- Keep responses brief (2-3 sentences)
- Use professional CTI analyst language

## Configuration

### Local Development (without Docker)

Set the environment variable:
```bash
export AI_SERVICE_URL=http://localhost:11434
```

### Docker Deployment

Add to `docker-compose.yml` file:
```yaml
environment:
  AI_SERVICE_URL: http://host.docker.internal:11434
```

When running inside Docker, the system automatically maps `localhost:11434` to `host.docker.internal:11434`.

## Example AI Service Implementation

### Python/Flask Example

```python
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/analyze', methods=['POST'])
def analyze():
    data = request.json
    
    title = data.get('title')
    content = data.get('content')
    category = data.get('category')
    criticality_score = data.get('criticality_score')
    analysis = f"""
    This entry was classified as '{category}' category with a criticality score of {criticality_score}/100.
    The content suggests {category.lower()} activity, which aligns with the assigned category.
    The criticality score reflects the potential impact level based on current threat indicators.
    """
    
    return jsonify({
        "analysis": analysis.strip()
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=11434)
```

### Ollama Integration Example

If you're using Ollama directly, you'll need a wrapper service that:
1. Receives the `/analyze` request
2. Formats a prompt for Ollama
3. Calls Ollama's `/api/generate` endpoint
4. Returns the response in the expected format

## Error Handling

- If the AI service is unavailable, the system continues normally
- Entry creation is **never** blocked by the AI service
- AI analysis is added asynchronously after entry creation
- Errors are logged but do not affect user experience

## Testing

Test your AI service:

```bash
curl -X POST http://localhost:11434/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Ransomware Attack Detected",
    "content": "A new ransomware variant has been discovered...",
    "category": "Malware Analysis",
    "criticality_score": 85
  }'
```

Expected response:
```json
{
  "analysis": "This entry was classified as 'Malware Analysis' category with a criticality score of 85/100..."
}
```

## Security Notes

- AI service runs locally (not exposed to the internet)
- Sensitive data should not be sent to external AI services
- AI analysis is stored in the database but is optional
- System works completely without AI service
