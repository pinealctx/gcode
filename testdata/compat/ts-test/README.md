# TS Runtime Tests

Verify the generated `.pb.ts` files are valid TypeScript and export correct runtime values.

## Prerequisites

- Node.js (v18+)
- npm

## Run

```bash
cd testdata/compat/ts-test

# Install dependencies (first time only)
npm install

# Type check — tsc --noEmit on all generated files
npm run typecheck

# Runtime tests — tsx test.ts (enum values, name mapping, validation rules, cross-file imports)
npm test
```

## What it tests

- **Enum values**: numeric constants match proto definitions
- **Enum name mapping**: `StatusName[value]` returns the correct string
- **Validation rules**: field constraints (minLength, maxLength, format, etc.) are exported correctly
- **Interface type safety**: generated interfaces compile and accept valid data
- **Cross-file imports**: types imported from other `.pb.ts` files resolve correctly
