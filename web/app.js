const MAGIC_HEADER = "ENVB1";
const SALT_SIZE = 16;
const NONCE_SIZE = 12;
const PBKDF2_ITERATIONS = 600000;

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const magicBytes = encoder.encode(MAGIC_HEADER);

const fileInput = document.getElementById("fileInput");
const cipherInput = document.getElementById("cipherInput");
const passwordInput = document.getElementById("passwordInput");
const decryptBtn = document.getElementById("decryptBtn");
const clearBtn = document.getElementById("clearBtn");
const copyBtn = document.getElementById("copyBtn");
const downloadBtn = document.getElementById("downloadBtn");
const statusEl = document.getElementById("status");
const outputEl = document.getElementById("output");

let currentJson = "";

decryptBtn.addEventListener("click", async () => {
  setBusy(true);
  setStatus("Decrypting...", "");

  try {
    const password = passwordInput.value;
    if (!password) {
      throw new Error("Password is required.");
    }

    const sourceBytes = await getInputBytes();
    const plaintextBytes = await decryptPayload(sourceBytes, password);
    const plaintext = decoder.decode(plaintextBytes);

    const parsed = JSON.parse(plaintext);
    currentJson = JSON.stringify(parsed, null, 2);
    outputEl.textContent = currentJson;
    setStatus("Decryption completed.", "success");
  } catch (error) {
    currentJson = "";
    outputEl.textContent = "No data yet.";
    setStatus(error instanceof Error ? error.message : String(error), "error");
  } finally {
    setBusy(false);
  }
});

clearBtn.addEventListener("click", () => {
  fileInput.value = "";
  cipherInput.value = "";
  passwordInput.value = "";
  currentJson = "";
  outputEl.textContent = "No data yet.";
  setStatus("", "");
});

copyBtn.addEventListener("click", async () => {
  if (!currentJson) {
    setStatus("Nothing to copy.", "error");
    return;
  }
  try {
    await navigator.clipboard.writeText(currentJson);
    setStatus("Copied to clipboard.", "success");
  } catch {
    setStatus("Clipboard permission denied.", "error");
  }
});

downloadBtn.addEventListener("click", () => {
  if (!currentJson) {
    setStatus("Nothing to download.", "error");
    return;
  }

  const blob = new Blob([currentJson], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "env.snapshot.json";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  setStatus("JSON downloaded.", "success");
});

async function getInputBytes() {
  const text = cipherInput.value.trim();
  if (text) {
    return decodeBase64Text(text);
  }

  const file = fileInput.files && fileInput.files[0];
  if (!file) {
    throw new Error("Upload a file or paste base64 content.");
  }

  const raw = new Uint8Array(await file.arrayBuffer());
  if (hasMagicHeader(raw)) {
    return raw;
  }

  const asText = decoder.decode(raw).trim();
  if (!asText) {
    throw new Error("Uploaded file is empty.");
  }
  return decodeBase64Text(asText);
}

function decodeBase64Text(input) {
  const compact = input
    .replace(/\s+/g, "")
    .replace(/-/g, "+")
    .replace(/_/g, "/");

  if (!compact) {
    throw new Error("Base64 input is empty.");
  }

  const padding = (4 - (compact.length % 4)) % 4;
  const normalized = compact + "=".repeat(padding);

  let decoded;
  try {
    const binary = atob(normalized);
    decoded = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      decoded[i] = binary.charCodeAt(i);
    }
  } catch {
    throw new Error("Invalid base64 input.");
  }

  if (!hasMagicHeader(decoded)) {
    throw new Error("Decoded data is not a valid ciphertext (missing ENVB1 header).");
  }
  return decoded;
}

function hasMagicHeader(bytes) {
  if (bytes.length < magicBytes.length) {
    return false;
  }
  for (let i = 0; i < magicBytes.length; i += 1) {
    if (bytes[i] !== magicBytes[i]) {
      return false;
    }
  }
  return true;
}

async function decryptPayload(ciphertext, password) {
  const minLength = magicBytes.length + SALT_SIZE + NONCE_SIZE + 16;
  if (ciphertext.length < minLength) {
    throw new Error("Ciphertext is too short.");
  }

  if (!hasMagicHeader(ciphertext)) {
    throw new Error("Invalid header. This data was not produced by the tool.");
  }

  const saltStart = magicBytes.length;
  const saltEnd = saltStart + SALT_SIZE;
  const nonceEnd = saltEnd + NONCE_SIZE;

  const salt = ciphertext.slice(saltStart, saltEnd);
  const nonce = ciphertext.slice(saltEnd, nonceEnd);
  const body = ciphertext.slice(nonceEnd);

  const passwordKey = await crypto.subtle.importKey(
    "raw",
    encoder.encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );

  const aesKey = await crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt,
      iterations: PBKDF2_ITERATIONS,
      hash: "SHA-256",
    },
    passwordKey,
    {
      name: "AES-GCM",
      length: 256,
    },
    false,
    ["decrypt"]
  );

  try {
    const plaintext = await crypto.subtle.decrypt(
      {
        name: "AES-GCM",
        iv: nonce,
        additionalData: magicBytes,
        tagLength: 128,
      },
      aesKey,
      body
    );

    return new Uint8Array(plaintext);
  } catch {
    throw new Error("Decryption failed. Password is wrong or data is damaged.");
  }
}

function setStatus(message, type) {
  statusEl.textContent = message;
  statusEl.className = "status";
  if (type) {
    statusEl.classList.add(type);
  }
}

function setBusy(busy) {
  decryptBtn.disabled = busy;
  decryptBtn.textContent = busy ? "Decrypting..." : "Decrypt";
}
