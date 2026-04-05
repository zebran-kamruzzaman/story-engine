# Story Engine

A quiet desktop writing environment for long-form fiction. Story Engine tracks your narrative — characters, scenes, structure — without interrupting the work of writing.

---

## What it is

Story Engine is a local-first application. Your prose lives as plain `.md` files on your machine. There is no cloud sync, no account, no subscription. If the application were deleted tomorrow, your writing would be exactly where you left it.

The interface is a single window: a scene list on the left, an editor in the center, and a Mirror panel on the right. The Mirror panel is the core feature — it reads your writing and reflects back what it sees: who is in a scene, what a scene is about, and answers to questions you have about your own story.

---

## Features

### Writing environment
- Distraction-free prose editor built on CodeMirror 6
- Plain Markdown — no formatting toolbar, no rich text
- Autosaves 800ms after you stop typing, atomically (temp file then rename — your writing is never partially written)
- Word count per scene and project total
- Save indicator in the status bar

### Projects (Stories)
- Create multiple projects — each project is a story
- Open Story / New Story from the menu bar
- Each project is a folder in `~/Documents/StoryEngine/` containing your scenes and all generated data
- Switch projects without restarting

### Scenes
- Scenes are the parts of a story: chapters, lore fragments, character journals, anything
- Create, rename, delete scenes
- Drag-to-reorder: grab the ⠿ handle and move scenes into any order
- Double-click the scene title in the status bar to rename inline

### Mirror panel — three zones

**Characters** — project-wide roster built from all scenes by the LLM. Each character is collapsible to show a 2–3 sentence description of who they are, inferred from your prose. Click 🧠 to build or refresh the roster. The roster does not change when you switch scenes — it belongs to the whole story.

**Scene Summary** — a 2–4 sentence summary of whichever scene is currently open, written by the LLM. Changes when you switch scenes. Regenerates automatically after you write approximately 100 new words, or on demand with 🧠.

**Ask the Story** — a chat interface that answers questions about your story. Ask who a character was talking to in a specific location, what happened in an earlier scene, or anything else. The assistant ranks your scene files by relevance to the question and draws its answer from those files. Source scenes are listed below each response. Chat history persists per project.

### LLM integration
- Defaults to Ollama running locally — no API key needed, no data leaves your machine
- Configurable: change the endpoint and model in the ⚙ settings to use any OpenAI-compatible provider (OpenAI, Groq, LM Studio, etc.)
- The LLM only runs when you ask it to or when the 100-word threshold is crossed — it never interrupts while you write

---

## Tech stack

| Layer | Technology |
|---|---|
| Desktop framework | Wails v3 (alpha.74) |
| Backend | Go 1.26 |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no GCC) |
| Frontend framework | React 19 |
| Language | TypeScript 5.6, strict mode |
| Styling | Tailwind CSS v4 |
| Editor | CodeMirror 6 |
| State | Zustand 5 |
| Drag-to-reorder | @hello-pangea/dnd |
| Build tool | Vite 6 |

---

## Prerequisites

### 1. Go 1.26+
```bash
go version
# go version go1.26.x windows/amd64
```
Download: https://go.dev/dl/

### 2. Node.js 20+
```bash
node --version
# v20.x.x
```
Download: https://nodejs.org/

### 3. Wails v3 CLI
```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails version
# v3.x.x
```
Add `%USERPROFILE%\go\bin` to PATH on Windows if `wails3` is not found after install.

### 4. WebView2 (Windows)
Ships with Windows 11 and modern Windows 10. If missing: https://developer.microsoft.com/en-us/microsoft-edge/webview2/

### 5. Ollama (for LLM features)
```bash
# Install from https://ollama.com/download
ollama pull llama3.2:3b
ollama serve
```
Ollama runs at `http://localhost:11434` by default. Story Engine is pre-configured to use it. LLM features are optional — the editor and scene management work without it.

> **No GCC required.** Story Engine uses `modernc.org/sqlite`, a pure-Go SQLite port. You do not need MinGW or any C toolchain.

---

## Running in development

```bash
# Clone and enter the project
git clone <repo-url>
cd story-engine

# Run the development build
wails3 dev
```

This starts the Go backend and the Vite dev server simultaneously. Changes to Go files trigger a backend rebuild. Changes to TypeScript and CSS hot-reload instantly.

First launch creates `~/Documents/StoryEngine/default/` automatically.

---

## Building for production

```bash
wails3 build
```

Produces a single self-contained executable in `bin/story-engine.exe` (Windows). The React frontend is compiled and embedded — no Node.js or separate web server is needed to run the binary.

---

## LLM provider options

Story Engine uses the OpenAI chat completions protocol (`POST /v1/chat/completions`). Any compatible provider works by changing the endpoint and model in ⚙ Settings.

| Provider | Endpoint | API Key | Notes |
|---|---|---|---|
| Ollama (default) | `http://localhost:11434/v1` | None | Runs locally |

Changes take effect immediately — no restart needed.

### Recommended local models (Ollama)

```bash
ollama pull llama3.2:3b    # 2.0 GB — default, runs on 8 GB RAM
ollama pull phi3:mini       # 2.3 GB — very efficient
ollama pull qwen2.5:3b      # 1.9 GB — good at structured output
ollama pull mistral:7b      # 4.1 GB — stronger analysis
```

---

## Troubleshooting

**App opens but the editor is blank after clicking a scene**
Open DevTools (right-click → Inspect → Console). If you see a `GetSceneContent` error, the scene file may have been moved or deleted from `scenes/`. The startup sync will remove the orphaned DB record on next launch.

**LLM features return "is Ollama running?"**
Run `ollama serve` in a terminal. On Windows, check the system tray for the Ollama icon — it should start automatically at login.

**"Could not reach the LLM" with a cloud provider**
Verify the endpoint URL has no trailing slash and the API key is correct. Test with:
```bash
curl -H "Authorization: Bearer <your-key>" https://api.openai.com/v1/models
```

**Scenes appear in the wrong order after restart**
The order is stored in the SQLite cache. If the cache is missing or corrupt, scenes fall back to filesystem order. Delete `.engine/project.db` and restart — the startup sync will rebuild it.

**Build fails with `pattern all:frontend/dist: no matching files found`**
The frontend hasn't been built yet. Run `wails3 dev` once (which builds the frontend) before attempting `wails3 build`.
