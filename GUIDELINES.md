# Guidelines on how to write code for the Apex Course

Follow these guidelines when writing code in this repository.

## Writing code

Every piece of code should be written in a separate branch and merged into `main` after review. 

### Creating branches

The branches should be named like `{component}/{type}/{description}`.

- `component`: The component of the infrastructure you're working on. This can be one of `server`, `embedded`.
- `type`: The type of code you're writing. This can be one of `feature`, `bugfix`, `refactor`, `test`, `enhancement`.
- `description`: A brief description of the functionality of the code you're writing.

Example: `server/feature/new-endpoint`

### Creating a pull request

When you create a branch that eventually will be merged into `main`, you should create a pull request. Then, after a review, the branch can be merged into `main`.