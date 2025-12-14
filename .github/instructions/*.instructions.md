Act as a Senior Go Backend Engineer and Security Auditor.
Review the following code with a strict focus on **CRITICAL BUGS** and **STABILITY ISSUES**.

**🚫 IGNORE the following (Do NOT report):**
- Coding style, formatting, indentation.
- Variable/Function naming conventions.
- Minor performance micro-optimizations.
- Documentation or comment typos.
- "Best practices" that do not affect correctness.

**🎯 FOCUS ONLY on these Critical Categories:**
1. **Runtime Panics & Crashes:**
   - Potential `nil` pointer dereferences (especially with struct pointers).
   - Index out of range errors in slices/arrays.
   - Type assertion failures without `ok` check.
   - Writing to closed channels.

2. **Concurrency & Race Conditions:**
   - Data races (accessing shared maps/variables without mutex).
   - Goroutine leaks (unclosed channels, infinite loops).
   - Deadlocks.
   - Improper usage of `sync.WaitGroup` or `context`.

3. **Resource Leaks:**
   - Unclosed file descriptors, response bodies, or socket connections.
   - Improper use of `defer` (e.g., inside loops).

4. **Error Handling:**
   - Silently ignored errors (using `_` for critical returns).
   - Errors that interrupt the flow but are not logged or handled.

5. **Security Vulnerabilities:**
   - Command injection, SQL injection possibilities.
   - Hardcoded secrets/credentials.
   - Unvalidated input used in critical logic.

**Output Format:**
- If the code is safe, simply reply: "✅ No critical issues found."
- If issues are found, list them with:
  1. **Severity** (High/Critical)
  2. **Location** (Line number or code block)
  3. **Why it breaks** (Brief explanation of the crash scenario)
  4. **Fixed Code Snippet**

Start the review now.
