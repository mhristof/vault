package builtinplugins

import (
	"github.com/hashicorp/vault/logical/plugin"
	"github.com/hashicorp/vault/plugins/database/cassandra"
	"github.com/hashicorp/vault/plugins/database/hana"
	"github.com/hashicorp/vault/plugins/database/mongodb"
	"github.com/hashicorp/vault/plugins/database/mssql"
	"github.com/hashicorp/vault/plugins/database/mysql"
	"github.com/hashicorp/vault/plugins/database/postgresql"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
)

var databasePlugins = map[string]BuiltinFactory{
	// These four databasePlugins all use the same mysql implementation but with
	// different username settings passed by the constructor.
	"mysql-database-plugin":        mysql.New(mysql.MetadataLen, mysql.MetadataLen, mysql.UsernameLen),
	"mysql-aurora-database-plugin": mysql.New(credsutil.NoneLength, mysql.LegacyMetadataLen, mysql.LegacyUsernameLen),
	"mysql-rds-database-plugin":    mysql.New(credsutil.NoneLength, mysql.LegacyMetadataLen, mysql.LegacyUsernameLen),
	"mysql-legacy-database-plugin": mysql.New(credsutil.NoneLength, mysql.LegacyMetadataLen, mysql.LegacyUsernameLen),

	"postgresql-database-plugin": postgresql.New,
	"mssql-database-plugin":      mssql.New,
	"cassandra-database-plugin":  cassandra.New,
	"mongodb-database-plugin":    mongodb.New,
	"hana-database-plugin":       hana.New,
}

// BuiltinFactory is the func signature that should be returned by
// the plugin's New() func.
type BuiltinFactory func() (interface{}, error)

// Get returns the BuiltinFactory func for a particular backend plugin
// from the databasePlugins map.
func Get(name string, pluginType plugin.PluginType) (BuiltinFactory, bool) {
	searchMap := make(map[string]BuiltinFactory)
	switch pluginType {
	case plugin.PluginTypeCredential:
		searchMap = credentialBackends
	case plugin.PluginTypeSecrets:
		searchMap = logicalBackends
	case plugin.PluginTypeDatabase:
		searchMap = databasePlugins
	}
	f, ok := searchMap[name]
	return f, ok
}

// TODO should this now include more keys?
// Keys returns the list of plugin names that are considered builtin databasePlugins.
func Keys() []string {
	keys := make([]string, len(databasePlugins))

	i := 0
	for k := range databasePlugins {
		keys[i] = k
		i++
	}

	return keys
}
