const ANSI_RE = /\x1b\[[^A-Za-z]*[A-Za-z]/g

/**
 * Merge a new chunk into accumulated text, simulating terminal behavior:
 * - ANSI escape codes are stripped
 * - \r (carriage return) overwrites the current line from the start
 */
export function applyChunk(prev: string, chunk: string): string {
  const clean = chunk.replace(ANSI_RE, "")
  const combined = prev + clean
  return combined
    .split("\n")
    .map((line) => {
      const parts = line.split("\r")
      return parts[parts.length - 1]
    })
    .join("\n")
}
