# JSON Type Checker

This project aims to solve IDE support for typed JSON files. By defining a `[name].typedef.json`-file, `[name].json` can be type checked.

## Usage

1. Download `jtc` from the releases for this repo.
2. Make `jtc` executable
3. Run `jtc --directory /path/to/json/files/parent/dir`.

The errors will be printed to console. If nothing is printed, no issues was found.

Here is an example:
```sh
$ ./jtc --directory test_files/
[JTC]: 📜 Validating test_files/test.json
[JTC]: ❌ Object missing key 'command' at .jobs[0]
[JTC]: ❌ Expected number at .jobs[0].run_id
[JTC]: ❌ Object missing key 'run_id' at .jobs[1]
[JTC]: ✅ Successfully validated test_files/test.json
```

## Typedefinition

Each field is named a "Node". The root node will contain child nodes. There are differeent types of nodes:
- `number`
- `string`
- `list`
- `object`

### `number` and `string`

These nodes has no additional properties: 

```json
{
  "type": "string"
}
```

```json
{
  "type": "number"
}
```

### `list`

A list needs to specify how its children is going to look:


```json
{
  "type": "list",
  "children": {
    // Specify how child node here
  }
}
```

### `object`

An object needs to specify what properties it has and how it's child nodes will look:


```json
{
  "type": "object",
  "property": {
    "somePropertyName1": {
      // Specify how child node here
    },
    "somePropertyName2": {
      // Specify how child node here
    },
    "somePropertyName3": {
      // Specify how child node here
    }
  }
}
```

### Custom types

Custom types can be defined in the "definition"-field as seen in this example. The definition will be available for the current node and all child nodes.

```json
{
  "type": "...",
  // ...
  "define": {
    "person": {
      "type": "object",
      "properties": {
        "firstName": {
          "type": "string"
        },
        "friend": {
          "type": "person",
          "optional": true
        }
      }
    }
  }
}
```

### Typedef Example
```json
{
    "type": "object",
    "properties": {
      "jobs": {
        "type": "list",
        "children": {
          "type": "object",
          "properties": {
            "name": {
              "type": "string"
            },
            "command": {
              "type": "string"
            },
            "tag": {
              "type": "string"
            },
            "run_id": {
              "type": "number"
            }
          }
        }
      }
    }
  }
```
