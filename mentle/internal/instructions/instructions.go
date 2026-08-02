package instructions

var InitInstructions = "# MemPalace Init Instructions\n\nSet up a new memory palace for your AI project.\n\n## Usage\n\n    mempalace-go init [directory]\n\n## Options\n- **directory** (optional): Path where the palace will be created. Defaults to configured palace path.\n\n## Setup Steps\n1. Run \"init\" to create the palace directory structure\n2. The palace will be initialized with a WAL (Write-Ahead Log) directory\n3. The ONNX embedding model will be auto-downloaded on first use\n\n## Example\n\n    mempalace-go init ~/my-project/.mempalace\n\nThis creates a memory palace at the specified path.\n"

var SearchInstructions = "# MemPalace Search Instructions\n\nSearch your memory palace for relevant information.\n\n## Usage\n\n    mempalace-go search [query]\n\n## Options\n- **query** (required): The search query string\n\n## How It Works\n1. Your query is converted to a vector embedding using hugot (ONNX runtime)\n2. The vector store is searched for similar memories\n3. Results are returned with their wing and room location\n\n## Example\n\n    mempalace-go search \"how to configure the database\"\n\n## Tips\n- Be specific in your queries for better results\n- Include context like file names or function names\n- Results show [wing/room] followed by content preview\n"

var MineInstructions = "# MemPalace Mine Instructions\n\nMine project files and conversations into your memory palace.\n\n## Usage\n\n    mempalace-go mine [directory]\n\n## Options\n- **directory** (required): Path to the project directory to mine\n\n## What Gets Mined\n- Source code files (.go, .py, .js, .ts, etc.)\n- Documentation files\n- Conversation logs and transcripts\n- Configuration files\n\n## How It Works\n1. Files are scanned and processed by entity detectors\n2. Content is extracted and normalized\n3. Memories are stored in the vector database\n4. Each memory is tagged with wing and room metadata\n\n## Example\n\n    mempalace-go mine ./my-project\n\n## Tips\n- Mine regularly to keep your palace up to date\n- Mine after significant changes to your project\n- Use specific directories for focused mining\n"

var HelpInstructions = "# MemPalace CLI Commands\n\nAll available commands in mempalace-go:\n\n## Core Commands\n\n### init\n\n    mempalace-go init [directory]\n\nInitialize a new memory palace.\n\n### mine\n\n    mempalace-go mine [directory]\n\nMine project files into the palace.\n\n### search\n\n    mempalace-go search [query]\n\nSearch the memory palace.\n\n### status\n\n    mempalace-go status\n\nShow palace status.\n\n### wake-up\n\n    mempalace-go wake-up\n\nShow L0 + L1 context.\n\n## Utility Commands\n\n### repair\n\n    mempalace-go repair\n\nRebuild palace vector index.\n\n### compress\n\n    mempalace-go compress [wing]\n\nCompress drawers using AAAK Dialect.\n\n### split\n\n    mempalace-go split [directory]\n\nSplit mega transcript files.\n\n### hook\n\n    mempalace-go hook [action]\n\nRun hook logic.\n\n### instructions\n\n    mempalace-go instructions [init|search|mine|help|status]\n\nShow instructions for a specific command.\n\n### mcp\n\n    mempalace-go mcp\n\nStart the MCP server.\n"

var StatusInstructions = "# MemPalace Status Instructions\n\nCheck the status of your memory palace.\n\n## Usage\n\n    mempalace-go status\n\n## What It Shows\n- **Palace path**: Where your memory palace is stored\n- **Collection name**: The name of your memory collection\n\n## Example Output\n\n    Palace: ~/.mempalace\n    Collection: my-project\n\n## Tips\n- Verify your palace is set up correctly\n- Check that the palace path is accessible\n- Ensure you have write permissions to the palace directory\n"

func GetInstruction(name string) (string, bool) {
	switch name {
	case "init":
		return InitInstructions, true
	case "search":
		return SearchInstructions, true
	case "mine":
		return MineInstructions, true
	case "help":
		return HelpInstructions, true
	case "status":
		return StatusInstructions, true
	default:
		return "", false
	}
}
