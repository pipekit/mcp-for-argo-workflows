# Script Template Type

Execute inline scripts without needing custom container images.

## Key Fields

- **image** (required) - Container image with interpreter
- **source** (required) - Inline script code
- **command** - Script interpreter (e.g., python, bash)

## Example

```yaml
templates:
  - name: gen-random
    script:
      image: python:alpine
      command: [python]
      source: |
        import random
        result = random.randint(1, 100)
        print(result)
```

## When to Use

Script templates are ideal when you need to run inline code without building a custom container image.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.