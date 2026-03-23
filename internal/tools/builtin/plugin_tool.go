package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/plugin"
	"github.com/meowai/blackcat/internal/tools"
)

// PluginTool exposes plugin management to the agent via chat.
type PluginTool struct {
	manager *plugin.Manager
}

// NewPluginTool creates a tool backed by the given plugin manager.
func NewPluginTool(manager *plugin.Manager) *PluginTool {
	return &PluginTool{manager: manager}
}

// Info returns the tool definition for manage_plugins.
func (t *PluginTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "manage_plugins",
		Description: "Manage BlackCat plugins: search, install, uninstall, start, stop, list, info, configure.",
		Category:    "plugin",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "Action to perform",
				Required:    true,
				Enum:        []string{"search", "install", "uninstall", "start", "stop", "list", "info", "configure"},
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "Plugin name (author/name format)",
				Required:    false,
			},
			{
				Name:        "version",
				Type:        "string",
				Description: "Plugin version (semver)",
				Required:    false,
			},
			{
				Name:        "type",
				Type:        "string",
				Description: "Plugin type filter (provider, channel, domain, scanner, hook)",
				Required:    false,
			},
			{
				Name:        "command",
				Type:        "string",
				Description: "Plugin binary command (for install)",
				Required:    false,
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query string",
				Required:    false,
			},
			{
				Name:        "config",
				Type:        "string",
				Description: "Config as key=value pairs separated by commas",
				Required:    false,
			},
		},
	}
}

// Execute runs the requested plugin management action.
func (t *PluginTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return tools.Result{
			Error:    "action parameter is required",
			ExitCode: 1,
		}, nil
	}

	switch action {
	case "list":
		return t.executeList(args)
	case "install":
		return t.executeInstall(args)
	case "uninstall":
		return t.executeUninstall(args)
	case "start":
		return t.executeStart(args)
	case "stop":
		return t.executeStop(args)
	case "info":
		return t.executeInfo(args)
	case "configure":
		return t.executeConfigure(args)
	case "search":
		return t.executeSearch(args)
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action: %s", action),
			ExitCode: 1,
		}, nil
	}
}

func (t *PluginTool) executeList(args map[string]any) (tools.Result, error) {
	var plugins []plugin.PluginInstance

	if pt, ok := args["type"].(string); ok && pt != "" {
		plugins = t.manager.ListByType(plugin.PluginType(pt))
	} else {
		plugins = t.manager.List()
	}

	if len(plugins) == 0 {
		return tools.Result{Output: "No plugins installed."}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Installed plugins (%d):\n", len(plugins)))
	for _, p := range plugins {
		sb.WriteString(fmt.Sprintf("  - %s v%s [%s] (%s)\n",
			p.Manifest.Name, p.Manifest.Version, p.Manifest.Type, p.State))
	}
	return tools.Result{Output: sb.String()}, nil
}

func (t *PluginTool) executeInstall(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	version, _ := args["version"].(string)
	pluginType, _ := args["type"].(string)
	command, _ := args["command"].(string)

	if name == "" {
		return tools.Result{Error: "name is required for install", ExitCode: 1}, nil
	}

	manifest := plugin.PluginManifest{
		Name:         name,
		Version:      version,
		Type:         plugin.PluginType(pluginType),
		Command:      command,
		Protocol:     "jsonrpc",
		Capabilities: []string{},
	}

	if version == "" {
		manifest.Version = "0.0.0"
	}
	if command == "" {
		manifest.Command = name
	}

	if err := t.manager.Install(manifest); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("install failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{
		Output: fmt.Sprintf("Plugin %s v%s installed successfully.", manifest.Name, manifest.Version),
	}, nil
}

func (t *PluginTool) executeUninstall(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{Error: "name is required for uninstall", ExitCode: 1}, nil
	}

	if err := t.manager.Uninstall(name); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("uninstall failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Plugin %s uninstalled.", name)}, nil
}

func (t *PluginTool) executeStart(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{Error: "name is required for start", ExitCode: 1}, nil
	}

	if err := t.manager.Start(name); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("start failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Plugin %s started.", name)}, nil
}

func (t *PluginTool) executeStop(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{Error: "name is required for stop", ExitCode: 1}, nil
	}

	if err := t.manager.Stop(name); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("stop failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Plugin %s stopped.", name)}, nil
}

func (t *PluginTool) executeInfo(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{Error: "name is required for info", ExitCode: 1}, nil
	}

	inst, ok := t.manager.Get(name)
	if !ok {
		return tools.Result{
			Error:    fmt.Sprintf("plugin %q not found", name),
			ExitCode: 1,
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Plugin: %s\n", inst.Manifest.Name))
	sb.WriteString(fmt.Sprintf("Version: %s\n", inst.Manifest.Version))
	sb.WriteString(fmt.Sprintf("Type: %s\n", inst.Manifest.Type))
	sb.WriteString(fmt.Sprintf("State: %s\n", inst.State))
	sb.WriteString(fmt.Sprintf("Author: %s\n", inst.Manifest.Author))
	sb.WriteString(fmt.Sprintf("Description: %s\n", inst.Manifest.Description))
	sb.WriteString(fmt.Sprintf("Command: %s\n", inst.Manifest.Command))
	sb.WriteString(fmt.Sprintf("Protocol: %s\n", inst.Manifest.Protocol))
	if len(inst.Manifest.Capabilities) > 0 {
		sb.WriteString(fmt.Sprintf("Capabilities: %s\n", strings.Join(inst.Manifest.Capabilities, ", ")))
	}
	if inst.Port > 0 {
		sb.WriteString(fmt.Sprintf("Port: %d\n", inst.Port))
	}

	return tools.Result{Output: sb.String()}, nil
}

func (t *PluginTool) executeConfigure(args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{Error: "name is required for configure", ExitCode: 1}, nil
	}

	configStr, _ := args["config"].(string)
	if configStr == "" {
		return tools.Result{Error: "config parameter is required (key=value,key=value)", ExitCode: 1}, nil
	}

	cfg := make(map[string]string)
	for _, pair := range strings.Split(configStr, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			cfg[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if err := t.manager.Configure(name, cfg); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("configure failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Plugin %s configured.", name)}, nil
}

func (t *PluginTool) executeSearch(_ map[string]any) (tools.Result, error) {
	return tools.Result{
		Output: "Search requires a remote registry connection. Use 'blackcat plugin search <query>' from the CLI.",
	}, nil
}
