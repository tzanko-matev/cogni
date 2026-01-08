package runner

import "cogni/internal/agent"

// defaultToolDefinitions returns built-in tool definitions.
func defaultToolDefinitions() []agent.ToolDefinition {
	disallowExtras := agent.BoolPointer(false)
	return []agent.ToolDefinition{
		{
			Name:        "list_files",
			Description: "List files in the repository. The glob argument follows .gitignore rules. Equivalent to `rg --files -g {glob}`",
			Parameters: &agent.ToolSchema{
				Type: "object",
				Properties: map[string]agent.ToolSchema{
					"glob": agent.StringSchema(),
				},
				AdditionalProperties: disallowExtras,
			},
		},
		{
			Name:        "list_dir",
			Description: "List directory entries in the repository with depth limits and pagination",
			Parameters: &agent.ToolSchema{
				Type: "object",
				Properties: map[string]agent.ToolSchema{
					"path":   agent.StringSchema(),
					"offset": agent.IntegerSchema(),
					"limit":  agent.IntegerSchema(),
					"depth":  agent.IntegerSchema(),
				},
				Required:             []string{"path"},
				AdditionalProperties: disallowExtras,
			},
		},
		{
			Name:        "search",
			Description: "Search for a query string in files",
			Parameters: &agent.ToolSchema{
				Type: "object",
				Properties: map[string]agent.ToolSchema{
					"query": agent.StringSchema(),
					"paths": agent.ArraySchema(agent.StringSchema()),
				},
				Required:             []string{"query"},
				AdditionalProperties: disallowExtras,
			},
		},
		{
			Name:        "read_file",
			Description: "Read a file from the repository",
			Parameters: &agent.ToolSchema{
				Type: "object",
				Properties: map[string]agent.ToolSchema{
					"path":       agent.StringSchema(),
					"start_line": agent.IntegerSchema(),
					"end_line":   agent.IntegerSchema(),
				},
				Required:             []string{"path"},
				AdditionalProperties: disallowExtras,
			},
		},
	}
}
