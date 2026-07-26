/*
Package reproslice minimizes failing JSON payloads while preserving an
externally observed command failure.

Candidates are serialized as compact JSON and supplied to a shell command
through a temporary file referenced by the {input} placeholder. A candidate is
accepted only when it matches every configured failure condition.
*/
package reproslice
