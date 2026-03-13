# Lessons

- When refactoring shutdown or polling flows, propagate `context.Context` through every blocking I/O boundary and add a cancellation test before calling the change done.
