package test_connect

import (
	"context"
	"testing"

	cconf "github.com/pip-services3-gox/pip-services3-commons-gox/config"
	conn "github.com/pip-services3-gox/pip-services3-sqlserver-gox/connect"
	"github.com/stretchr/testify/assert"
)

func TestSqlServerConnectionResolver(t *testing.T) {

	dbConfig := cconf.NewConfigParamsFromTuples(
		"connection.host", "localhost",
		"connection.port", 1433,
		"connection.database", "test",
		"connection.encrypt", true,
		"credential.username", "sa",
		"credential.password", "pwd#123",
	)

	resolver := conn.NewSqlServerConnectionResolver()
	resolver.Configure(context.Background(), dbConfig)

	uri, err := resolver.Resolve(context.Background(), "")
	assert.Nil(t, err)

	assert.NotEmpty(t, uri)
	assert.Equal(t, "sqlserver://sa:pwd#123@localhost:1433/test?encrypt=true", uri)
}
