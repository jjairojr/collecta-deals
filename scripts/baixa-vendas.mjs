#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { appendFileSync, existsSync, mkdirSync, openSync, closeSync, readFileSync, rmdirSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const dir = join(root, "data", "baixa-vendas");
const lock = join(dir, ".lock");
const topic = process.env.NTFY_TOPIC || "";

const pad = (n) => String(n).padStart(2, "0");
const now = new Date();
const stamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}`;
const report = join(dir, `${stamp}.md`);
const logPath = join(dir, `${stamp}.log`);

mkdirSync(dir, { recursive: true });

const runLog = (msg) => appendFileSync(join(dir, "runs.log"), `${stamp} ${msg}\n`);

async function push(title, body) {
  if (!topic) return;
  try {
    const res = await fetch(`https://ntfy.sh/${topic}`, {
      method: "POST",
      headers: { Title: encodeURIComponent(title), Tags: "package" },
      body,
    });
    if (!res.ok) runLog(`ntfy falhou: HTTP ${res.status}`);
  } catch (err) {
    runLog(`ntfy falhou: ${err.message}`);
  }
}

if (existsSync(lock) && Date.now() - statSync(lock).mtimeMs > 3 * 60 * 60 * 1000) {
  rmdirSync(lock);
}
try {
  mkdirSync(lock);
} catch {
  runLog("outra execução em andamento");
  process.exit(0);
}

try {
  const chrome = spawnSync("pgrep", ["-xq", "Google Chrome"]);
  if (chrome.status !== 0) {
    runLog("chrome fechado");
    await push("Baixa de vendas", "Pulada: Chrome fechado");
  } else {
    const fd = openSync(logPath, "a");
    const run = spawnSync(
      join(homedir(), ".local", "bin", "claude"),
      [
        "-p",
        `/baixa-vendas --auto ${stamp}`,
        "--chrome",
        "--permission-mode",
        "acceptEdits",
        "--allowedTools",
        "Bash",
        "Read",
        "Write",
        "Edit",
        "mcp__claude-in-chrome__*",
      ],
      { cwd: root, stdio: ["ignore", fd, fd], timeout: 2 * 60 * 60 * 1000 },
    );
    closeSync(fd);
    const code = run.error ? run.error.message : run.status;
    runLog(`exit=${code}`);
    if (existsSync(report)) {
      await push("Baixa de vendas", readFileSync(report, "utf8").slice(0, 3000));
    } else {
      await push("Baixa de vendas falhou", `exit ${code}, sem relatório. Log: data/baixa-vendas/${stamp}.log`);
    }
  }
} finally {
  rmdirSync(lock);
}
