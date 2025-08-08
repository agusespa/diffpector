# Mock Files for Evaluation

This directory contains mock file content used during evaluation testing. The evaluator creates a simulated repository environment by:

1. **Extracting filenames** from diff headers (`+++ b/filename`)
2. **Loading mock content** from this directory using the exact same paths as in the diffs
3. **Falling back** to simple generated content if no mock file exists

## Directory Structure

The mock files mirror the paths referenced in the diff files:

```
mocks/
├── internal/
│   ├── database/
│   │   └── user.go          # Mock database operations
│   └── utils/
│       └── string.go        # Mock utility functions
└── README.md
```

## Adding New Mock Files

To add mock content for a new test case:

1. Create the file using the exact same path as referenced in your diff file
2. Include realistic code with the issues you want to test for
3. The evaluator will automatically use this content