package builtinplugins

import (
	"github.com/hashicorp/vault/builtin/plugin"

	ad "github.com/hashicorp/vault-plugin-secrets-ad/plugin"
	"github.com/hashicorp/vault-plugin-secrets-alicloud"
	azure "github.com/hashicorp/vault-plugin-secrets-azure"
	gcp "github.com/hashicorp/vault-plugin-secrets-gcp/plugin"
	kv "github.com/hashicorp/vault-plugin-secrets-kv"
	"github.com/hashicorp/vault/builtin/logical/aws"
	"github.com/hashicorp/vault/builtin/logical/cassandra"
	"github.com/hashicorp/vault/builtin/logical/consul"
	"github.com/hashicorp/vault/builtin/logical/database"
	"github.com/hashicorp/vault/builtin/logical/mongodb"
	"github.com/hashicorp/vault/builtin/logical/mssql"
	"github.com/hashicorp/vault/builtin/logical/mysql"
	"github.com/hashicorp/vault/builtin/logical/nomad"
	"github.com/hashicorp/vault/builtin/logical/pki"
	"github.com/hashicorp/vault/builtin/logical/postgresql"
	"github.com/hashicorp/vault/builtin/logical/rabbitmq"
	"github.com/hashicorp/vault/builtin/logical/ssh"
	"github.com/hashicorp/vault/builtin/logical/totp"
	"github.com/hashicorp/vault/builtin/logical/transit"

	credAliCloud "github.com/hashicorp/vault-plugin-auth-alicloud"
	credAzure "github.com/hashicorp/vault-plugin-auth-azure"
	credCentrify "github.com/hashicorp/vault-plugin-auth-centrify"
	credGcp "github.com/hashicorp/vault-plugin-auth-gcp/plugin"
	credJWT "github.com/hashicorp/vault-plugin-auth-jwt"
	credKube "github.com/hashicorp/vault-plugin-auth-kubernetes"
	credAppId "github.com/hashicorp/vault/builtin/credential/app-id"
	credAppRole "github.com/hashicorp/vault/builtin/credential/approle"
	credAws "github.com/hashicorp/vault/builtin/credential/aws"
	credCert "github.com/hashicorp/vault/builtin/credential/cert"
	credGitHub "github.com/hashicorp/vault/builtin/credential/github"
	credLdap "github.com/hashicorp/vault/builtin/credential/ldap"
	credOkta "github.com/hashicorp/vault/builtin/credential/okta"
	credRadius "github.com/hashicorp/vault/builtin/credential/radius"
	credUserpass "github.com/hashicorp/vault/builtin/credential/userpass"
)

var credentialBackends = map[string]BuiltinFactory{
	"alicloud":   toBuiltinFactory(credAliCloud.Factory),
	"app-id":     toBuiltinFactory(credAppId.Factory),
	"approle":    toBuiltinFactory(credAppRole.Factory),
	"aws":        toBuiltinFactory(credAws.Factory),
	"azure":      toBuiltinFactory(credAzure.Factory),
	"centrify":   toBuiltinFactory(credCentrify.Factory),
	"cert":       toBuiltinFactory(credCert.Factory),
	"gcp":        toBuiltinFactory(credGcp.Factory),
	"github":     toBuiltinFactory(credGitHub.Factory),
	"jwt":        toBuiltinFactory(credJWT.Factory),
	"kubernetes": toBuiltinFactory(credKube.Factory),
	"ldap":       toBuiltinFactory(credLdap.Factory),
	"okta":       toBuiltinFactory(credOkta.Factory),
	"radius":     toBuiltinFactory(credRadius.Factory),
	"userpass":   toBuiltinFactory(credUserpass.Factory),
}

var logicalBackends = map[string]BuiltinFactory{
	"ad":         toBuiltinFactory(ad.Factory),
	"alicloud":   toBuiltinFactory(alicloud.Factory),
	"aws":        toBuiltinFactory(aws.Factory),
	"azure":      toBuiltinFactory(azure.Factory),
	"cassandra":  toBuiltinFactory(cassandra.Factory),
	"consul":     toBuiltinFactory(consul.Factory),
	"database":   toBuiltinFactory(database.Factory),
	"gcp":        toBuiltinFactory(gcp.Factory),
	"kv":         toBuiltinFactory(kv.Factory),
	"mongodb":    toBuiltinFactory(mongodb.Factory),
	"mssql":      toBuiltinFactory(mssql.Factory),
	"mysql":      toBuiltinFactory(mysql.Factory),
	"nomad":      toBuiltinFactory(nomad.Factory),
	"pki":        toBuiltinFactory(pki.Factory),
	"plugin":     toBuiltinFactory(plugin.Factory),
	"postgresql": toBuiltinFactory(postgresql.Factory),
	"rabbitmq":   toBuiltinFactory(rabbitmq.Factory),
	"ssh":        toBuiltinFactory(ssh.Factory),
	"totp":       toBuiltinFactory(totp.Factory),
	"transit":    toBuiltinFactory(transit.Factory),
}

// TODO it's redundant to call this so many times
func toBuiltinFactory(ifc interface{}) BuiltinFactory {
	return func() (interface{}, error) {
		return ifc, nil
	}
}
