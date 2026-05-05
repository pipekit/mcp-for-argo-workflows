# Argo Workflows Expressions Reference

Expressions provide powerful data manipulation capabilities within Argo Workflows using the `{{=expression}}` syntax. Argo uses the `expr` expression language for evaluation.

## Expression Syntax

Expressions are enclosed in `{{=` and `}}`:

```yaml
value: "{{=inputs.parameters.count * 2}}"
```

## Accessing Variables in Expressions

Within expressions, variables are accessed without double curly braces:

| Variable Type | Template Syntax | Expression Syntax |
|---------------|-----------------|-------------------|
| Input parameter | `{{inputs.parameters.name}}` | `inputs.parameters.name` |
| Step output | `{{steps.stepname.outputs.result}}` | `steps.stepname.outputs.result` |
| Task output | `{{tasks.taskname.outputs.result}}` | `tasks.taskname.outputs.result` |
| Workflow param | `{{workflow.parameters.name}}` | `workflow.parameters.name` |
| Item (loop) | `{{item}}` | `item` |

## Arithmetic Operations

```yaml
# Addition
value: "{{=inputs.parameters.count + 1}}"

# Subtraction
value: "{{=inputs.parameters.total - inputs.parameters.used}}"

# Multiplication
value: "{{=inputs.parameters.count * 2}}"

# Division
value: "{{=inputs.parameters.total / inputs.parameters.parts}}"

# Modulo
value: "{{=inputs.parameters.index % 3}}"
```

## Comparison Operations

```yaml
# Equality
when: "{{=inputs.parameters.env == 'prod'}}"

# Not equal
when: "{{=inputs.parameters.status != 'skip'}}"

# Greater than / Less than
when: "{{=inputs.parameters.count > 10}}"
when: "{{=inputs.parameters.priority < 5}}"

# Greater/Less than or equal
when: "{{=inputs.parameters.count >= 0}}"
when: "{{=inputs.parameters.level <= 3}}"
```

## Logical Operations

```yaml
# AND
when: "{{=inputs.parameters.enabled == 'true' && inputs.parameters.env == 'prod'}}"

# OR
when: "{{=inputs.parameters.env == 'prod' || inputs.parameters.env == 'staging'}}"

# NOT
when: "{{=!(inputs.parameters.skip == 'true')}}"
```

## Conditional (Ternary) Expressions

```yaml
# condition ? true_value : false_value
value: "{{=inputs.parameters.env == 'prod' ? 'production' : 'development'}}"

# Nested conditionals
value: "{{=inputs.parameters.level == 'high' ? 1 : (inputs.parameters.level == 'medium' ? 2 : 3)}}"
```

## String Functions

### Basic String Operations

```yaml
# String concatenation
value: "{{='prefix-' + inputs.parameters.name + '-suffix'}}"

# Length
value: "{{=len(inputs.parameters.items)}}"

# Contains check
when: "{{=contains(inputs.parameters.tags, 'important')}}"
```

### String Manipulation with Sprig

Argo supports Sprig template functions. Use them with the `sprig` prefix:

```yaml
# Convert to upper/lower case
value: "{{=sprig.upper(inputs.parameters.name)}}"
value: "{{=sprig.lower(inputs.parameters.name)}}"

# Trim whitespace
value: "{{=sprig.trim(inputs.parameters.data)}}"
value: "{{=sprig.trimPrefix('prefix-', inputs.parameters.name)}}"
value: "{{=sprig.trimSuffix('-suffix', inputs.parameters.name)}}"

# Replace
value: "{{=sprig.replace('old', 'new', inputs.parameters.text)}}"

# Substring
value: "{{=sprig.substr(0, 10, inputs.parameters.name)}}"

# Split and join
value: "{{=sprig.split(',', inputs.parameters.csv)}}"
value: "{{=sprig.join(',', inputs.parameters.list)}}"

# Default value if empty
value: "{{=sprig.default('fallback', inputs.parameters.optional)}}"

# Quote strings
value: "{{=sprig.quote(inputs.parameters.message)}}"
```

## Numeric Functions

```yaml
# Type conversion
value: "{{=asInt(inputs.parameters.count)}}"
value: "{{=asFloat(inputs.parameters.price)}}"

# Math functions (via sprig)
value: "{{=sprig.max(inputs.parameters.a, inputs.parameters.b)}}"
value: "{{=sprig.min(inputs.parameters.a, inputs.parameters.b)}}"
value: "{{=sprig.floor(inputs.parameters.value)}}"
value: "{{=sprig.ceil(inputs.parameters.value)}}"
value: "{{=sprig.round(inputs.parameters.value, 2)}}"
```

## JSON Functions

### JSONPath Queries

```yaml
# Extract value using JSONPath
value: "{{=jsonpath(inputs.parameters.json_data, '$.items[0].name')}}"

# Multiple results
value: "{{=jsonpath(inputs.parameters.json_data, '$.items[*].name')}}"
```

### JSON Manipulation

```yaml
# Parse JSON string to object
value: "{{=fromJson(inputs.parameters.json_string)}}"

# Convert to JSON string
value: "{{=toJson(inputs.parameters.data)}}"

# Access JSON fields after parsing
value: "{{=fromJson(steps.get-config.outputs.result).database.host}}"
```

## Array/List Functions

```yaml
# Get array length
value: "{{=len(inputs.parameters.items)}}"

# Access by index
value: "{{=inputs.parameters.items[0]}}"

# Last element
value: "{{=inputs.parameters.items[len(inputs.parameters.items) - 1]}}"

# Sprig list functions
value: "{{=sprig.first(inputs.parameters.items)}}"
value: "{{=sprig.last(inputs.parameters.items)}}"
value: "{{=sprig.rest(inputs.parameters.items)}}"
value: "{{=sprig.initial(inputs.parameters.items)}}"
value: "{{=sprig.reverse(inputs.parameters.items)}}"
value: "{{=sprig.uniq(inputs.parameters.items)}}"
value: "{{=sprig.sortAlpha(inputs.parameters.items)}}"
```

## Map/Object Functions

```yaml
# Access map values
value: "{{=inputs.parameters.config['key']}}"
value: "{{=inputs.parameters.config.key}}"

# Check if key exists
when: "{{=hasKey(inputs.parameters.config, 'optional_field')}}"

# Get keys
value: "{{=sprig.keys(inputs.parameters.config)}}"

# Get values
value: "{{=sprig.values(inputs.parameters.config)}}"

# Merge maps
value: "{{=sprig.merge(inputs.parameters.defaults, inputs.parameters.overrides)}}"
```

## Date/Time Functions

```yaml
# Current timestamp
value: "{{=sprig.now()}}"
value: "{{=sprig.date('2006-01-02', sprig.now())}}"

# Format timestamp
value: "{{=sprig.dateModify('-24h', sprig.now())}}"

# Parse date strings
value: "{{=sprig.toDate('2006-01-02', inputs.parameters.date_string)}}"
```

## Status-Based Expressions

Common in `when` conditions:

```yaml
# Check step/task status
when: "{{=steps.previous.status == 'Succeeded'}}"
when: "{{=tasks.dependency.status == 'Failed'}}"

# Multiple status checks
when: "{{=steps.check.status == 'Succeeded' || steps.check.status == 'Skipped'}}"

# In onExit handlers
when: "{{=workflow.status == 'Failed'}}"
```

## Aggregation Functions

For handling outputs from parallel steps:

```yaml
# Aggregate results from fan-out
value: "{{=toJson(steps.parallel-step.outputs.result)}}"

# In DAG templates
value: "{{=toJson(tasks.fan-out.outputs.result)}}"
```

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expression-demo-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: count
      value: "10"
    - name: config
      value: '{"env": "prod", "replicas": 3}'

  templates:
  - name: main
    steps:
    - - name: calculate
        template: calculator
        arguments:
          parameters:
          - name: doubled
            value: "{{=asInt(workflow.parameters.count) * 2}}"
          - name: env
            value: "{{=fromJson(workflow.parameters.config).env}}"
    - - name: conditional
        template: worker
        when: "{{=fromJson(workflow.parameters.config).replicas > 1}}"
        arguments:
          parameters:
          - name: message
            value: "{{=steps.calculate.outputs.result == 'success' ? 'Proceeding' : 'Retrying'}}"

  - name: calculator
    inputs:
      parameters:
      - name: doubled
      - name: env
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo 'success'"]

  - name: worker
    inputs:
      parameters:
      - name: message
    container:
      image: alpine
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

## Expression vs Template Syntax

| Use Case | Template Syntax | Expression Syntax |
|----------|-----------------|-------------------|
| Simple substitution | `{{inputs.parameters.name}}` | Not needed |
| Arithmetic | Not supported | `{{=count + 1}}` |
| Conditionals | Limited | `{{=a > b ? 'yes' : 'no'}}` |
| JSON parsing | Not supported | `{{=fromJson(data).field}}` |
| String functions | Limited | `{{=sprig.upper(str)}}` |

## Common Gotchas

### Quoting Strings in Expressions

String literals in expressions need quotes:

```yaml
# Correct
when: "{{=inputs.parameters.env == 'prod'}}"

# Incorrect - will fail
when: "{{=inputs.parameters.env == prod}}"
```

### Type Coercion

Parameters are strings by default. Convert explicitly:

```yaml
# Convert string to int for arithmetic
value: "{{=asInt(inputs.parameters.count) + 1}}"

# Convert to float
value: "{{=asFloat(inputs.parameters.price) * 1.1}}"
```

### Nested Expressions

Expressions cannot be nested:

```yaml
# This will NOT work
value: "{{={{=inputs.parameters.nested}}}}"
```

### Empty String Handling

```yaml
# Check for empty string
when: "{{=inputs.parameters.optional != ''}}"

# Use default for empty values
value: "{{=sprig.default('fallback', inputs.parameters.optional)}}"
```

## See Also

- `argo://docs/variables` - Complete variable reference
- `argo://docs/parameters` - Parameter handling patterns
- `argo://examples/conditionals` - Conditional execution examples
