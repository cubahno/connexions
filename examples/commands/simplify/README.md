# Simplify Command Example

This example demonstrates what the `simplify` command removes from an OpenAPI spec.

## Files

- `source.yml` - Original spec with various union types and extensions
- `simplified.yml` - Generated simplified spec (after running `go generate`)

## Usage

```bash
cd examples/commands/simplify
go generate
```

Then compare `source.yml` and `simplified.yml` to see what was removed.

## What Gets Simplified

The `source.yml` contains examples of:

| Case | Source | Simplified |
|------|--------|------------|
| Required union (`anyOf`/`oneOf`) | Multiple variants | `anyOf`/`oneOf` removed |
| Optional union | Union property | Entire property removed |
| Many optional properties | All present | Limited to 5 (random selection) |

## Options Used

- `-output simplified.yml` - Write output to a file instead of stdout
- `-keep-optional 5` - Keep at most 5 optional properties per schema

