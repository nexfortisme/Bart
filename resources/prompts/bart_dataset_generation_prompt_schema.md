# Bart classifier dataset schema

Generate a JSON object with the following structure. Do not include comments or extra keys.

## Top-level keys

| Key | Type | Notes |
|-----|------|-------|
| `version` | string | Format `"MAJOR.MINOR"`, e.g. `"3.1"` |
| `description` | string | One sentence describing what this batch covers |
| `classifiers` | object | Contains `message_intent` and `tool_intent` |
| `edge_cases` | object | Contains `description` and `examples` |

---

## classifiers.message_intent

```json
"message_intent": {
  "description": "...",
  "classes": ["directed", "ambient"],
  "examples": [ ...message_intent_example ]
}
```

## classifiers.tool_intent

```json
"tool_intent": {
  "description": "...",
  "classes": ["weather", "time", "web_search", "web_fetch", "null"],
  "examples": [ ...tool_intent_example ]
}
```

---

## Example objects

All examples share these fields:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `text` | string | yes | Raw Discord message. Realistic casing, typos, slang — do not sanitize |
| `label` | string | yes | Must be a value from the parent classifier's `classes` array |
| `notes` | string | no | Brief rationale — required whenever the label is non-obvious |

**message_intent** labels: `"directed"` or `"ambient"` only.

**tool_intent** labels: `"weather"`, `"time"`, `"web_search"`, `"web_fetch"`, or `"null"` only.

---

## edge_cases

```json
"edge_cases": {
  "description": "...",
  "examples": [ ...edge_case_example ]
}
```

Edge case examples add one required field:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `classifier` | string | yes | `"message_intent"` or `"tool_intent"` |
| `text` | string | yes | Raw Discord message |
| `label` | string | yes | Correct label within that classifier's label space |
| `notes` | string | yes | Required — explain what makes this example ambiguous |

---

## Rules

- `null` in tool_intent means no tool is needed — the LLM can answer from context or training data
- Edge cases are messages likely to be misclassified — keyword overlap, implicit meaning, multi-intent, or Discord idioms that look like commands but aren't
- Every example's `label` must be a valid value from its classifier's `classes` list
- Do not add keys not listed above
- Output valid JSON only — no markdown fences, no comments
