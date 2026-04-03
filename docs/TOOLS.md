# Tools Reference

Tools are functions the agent can call autonomously during the agentic loop. They are exposed to the LLM as OpenAI-compatible function definitions.

## Core Tools

### shell
Execute a shell command with timeout and output capture.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | string | yes | Shell command to execute |
| `timeout` | number | no | Timeout in seconds (default: 30) |

### read_file
Read the contents of a file.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | File path to read |

### write_file
Write content to a file. Creates parent directories.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | File path to write |
| `content` | string | yes | Content to write |

### list_dir
List directory contents with file sizes and types.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory path |

### search_files
Search for files matching a pattern.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pattern` | string | yes | Glob or substring pattern |
| `path` | string | no | Root directory to search |

### http_request
Make HTTP requests (GET, POST, etc.)

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | Target URL |
| `method` | string | no | HTTP method (default: GET) |
| `body` | string | no | Request body |
| `headers` | object | no | Request headers |

### notify
Send a system notification.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | yes | Notification title |
| `body` | string | yes | Notification body |

## Memory Tools

### memory_store
Store a key-value pair in persistent memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Memory key |
| `value` | string | yes | Value to store |

### memory_recall
Retrieve a value by key.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Memory key to recall |

### memory_search
Search across all stored memory values.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |

### memory_list
List all keys in memory. No parameters.

### memory_forget
Remove a key from memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Key to remove |

## Ollama Tools

### ollama_list_models
List all models available in the local Ollama instance.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_url` | string | no | Ollama server URL (default: `http://127.0.0.1:11434`) |

### ollama_pull_model
Download a model from the Ollama registry.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `model` | string | yes | Model name (e.g. `llama3.2`, `qwen2.5-coder:7b`) |
| `base_url` | string | no | Ollama server URL (default: `http://127.0.0.1:11434`) |

### ollama_status
Check if Ollama is running and report available model count.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_url` | string | no | Ollama server URL (default: `http://127.0.0.1:11434`) |

## Android / MacDroid Tools

### device_list
List connected Android devices via ADB. No parameters.

### device_info
Get detailed device information.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `device_id` | string | no | Specific device (default: first found) |

### file_push
Push a local file to an Android device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `local_path` | string | yes | Local file path |
| `remote_path` | string | yes | Path on device |
| `device_id` | string | no | Specific device |

### file_pull
Pull a file from an Android device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `remote_path` | string | yes | Path on device |
| `local_path` | string | yes | Local destination |
| `device_id` | string | no | Specific device |

### file_list
List files on an Android device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory path on device |
| `device_id` | string | no | Specific device |

### app_list
List installed apps on device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `device_id` | string | no | Specific device |

### app_install
Install an APK on device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `apk_path` | string | yes | Path to APK file |
| `device_id` | string | no | Specific device |

### screenshot
Capture a screenshot from the device.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `output_path` | string | no | Local save path |
| `device_id` | string | no | Specific device |

### screen_record
Record the device screen.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `output_path` | string | no | Local save path |
| `duration` | number | no | Duration in seconds (default: 10) |
| `device_id` | string | no | Specific device |

## Network Tools

### network_scan
Scan local network for LLM servers (Ollama, llamafile).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subnet` | string | no | Subnet to scan (default: auto-detect) |

### network_info
Get local network interface information. No parameters.
