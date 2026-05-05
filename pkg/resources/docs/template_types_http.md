# HTTP Template Type

Make HTTP requests as workflow steps.

## Key Fields

- **url** (required) - URL to request
- **method** - HTTP method (default: GET)
- **headers** - HTTP headers
- **body** - Request body
- **timeoutSeconds** - Request timeout

## Example

```yaml
templates:
  - name: http-request
    http:
      url: "https://api.example.com/webhook"
      method: "POST"
      headers:
        - name: "Content-Type"
          value: "application/json"
      body: '{"message": "Hello from Argo"}'
      timeoutSeconds: 30
```

## When to Use

HTTP templates are ideal for calling REST APIs, webhooks, or external services.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.