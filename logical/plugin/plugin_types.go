package plugin

type PluginType int

const (
	PluginTypeCredential PluginType = iota
	PluginTypeSecrets
	PluginTypeDatabase
)
