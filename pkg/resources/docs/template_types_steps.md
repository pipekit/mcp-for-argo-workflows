# Steps Template Type

Define sequential execution with support for parallel step groups.

## Structure

Steps are organized into groups:
- Each group is a list of steps
- Steps within a group run in parallel
- Groups run sequentially

## Example

```yaml
templates:
  - name: hello-hello-hello
    steps:
      - - name: step1
          template: whalesay
          arguments:
            parameters: [{name: message, value: "hello1"}]
      - - name: step2a
          template: whalesay
          arguments:
            parameters: [{name: message, value: "hello2a"}]
        - name: step2b
          template: whalesay
          arguments:
            parameters: [{name: message, value: "hello2b"}]
```

## When to Use

Steps templates are ideal for simple sequential workflows with optional parallel groups.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.