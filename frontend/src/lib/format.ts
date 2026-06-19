// Pure formatters for the response view-mode switcher (raw / hex / base64).

// Unicode-safe base64. Encode to UTF-8 bytes first, then btoa in chunks so
// large bodies don't blow the String.fromCharCode call-stack.
export function toBase64(s: string): string {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  const CHUNK = 0x8000;
  for (let i = 0; i < bytes.length; i += CHUNK) {
    bin += String.fromCharCode(...bytes.subarray(i, i + CHUNK));
  }
  return btoa(bin);
}

// Classic hexdump: `offset  16 hex bytes  ascii`. Built once; rendered through
// CodeMirror which virtualizes, so size is fine.
export function toHex(s: string): string {
  const bytes = new TextEncoder().encode(s);
  const rows: string[] = [];
  for (let off = 0; off < bytes.length; off += 16) {
    const slice = bytes.subarray(off, off + 16);
    const hex: string[] = [];
    let ascii = "";
    for (let i = 0; i < 16; i++) {
      if (i < slice.length) {
        const b = slice[i];
        hex.push(b.toString(16).padStart(2, "0"));
        ascii += b >= 0x20 && b < 0x7f ? String.fromCharCode(b) : ".";
      } else {
        hex.push("  ");
      }
    }
    const offset = off.toString(16).padStart(8, "0");
    // group hex into two columns of 8 for readability
    const hexStr = hex.slice(0, 8).join(" ") + "  " + hex.slice(8).join(" ");
    rows.push(`${offset}  ${hexStr}  ${ascii}`);
  }
  return rows.join("\n");
}
