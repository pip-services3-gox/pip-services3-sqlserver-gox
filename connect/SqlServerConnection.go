package connect

import (
	"context"
	"database/sql"
	"math"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	cerr "github.com/pip-services3-gox/pip-services3-commons-gox/errors"

	cconf "github.com/pip-services3-gox/pip-services3-commons-gox/config"
	cref "github.com/pip-services3-gox/pip-services3-commons-gox/refer"
	clog "github.com/pip-services3-gox/pip-services3-components-gox/log"
)

// SqlServer connection using plain driver.
//
// By defining a connection and sharing it through multiple persistence components
// you can reduce number of used database connections.
//
//	Configuration parameters
//		- connection(s):
//			- discovery_key:        (optional) a key to retrieve the connection from IDiscovery
//			- host:                 host name or IP address
//			- port:                 port number (default: 27017)
//			- uri:                  resource URI or connection string with all parameters in it
//		- credential(s):
//			- store_key:            (optional) a key to retrieve the credentials from ICredentialStore
//			- username:             user name
//			- password:             user password
//		- options:
//			- connect_timeout:      (optional) number of milliseconds to wait before timing out when connecting a new client (default: 0)
//			- idle_timeout:         (optional) number of milliseconds a client must sit idle in the pool and not be checked out (default: 10000)
//			- max_pool_size:        (optional) maximum number of clients the pool should contain (default: 10)
//
//	References
//		- *:logger:*:*:1.0           (optional) ILogger components to pass log messages
//		- *:discovery:*:*:1.0        (optional) IDiscovery services
//		- *:credential-store:*:*:1.0 (optional) Credential stores to resolve credentials
type SqlServerConnection struct {
	defaultConfig *cconf.ConfigParams
	// The logger.
	Logger *clog.CompositeLogger
	// The connection resolver.
	ConnectionResolver *SqlServerConnectionResolver
	// The configuration options.
	Options *cconf.ConfigParams
	// The SqlServer connection pool object.
	Connection *sql.DB
	// The SqlServer database name.
	DatabaseName string

	retries int
}

const (
	DefaultConnectTimeout = 15000
	DefaultIdleTimeout    = 30000
	DefaultMaxPoolSize    = 3
	DefaultRetriesCount   = 3
)

// NewSqlServerConnection creates a new instance of the connection component.
func NewSqlServerConnection() *SqlServerConnection {
	c := &SqlServerConnection{
		defaultConfig: cconf.NewConfigParamsFromTuples(
			"options.connect_timeout", DefaultConnectTimeout,
			"options.idle_timeout", DefaultIdleTimeout,
			"options.max_pool_size", DefaultMaxPoolSize,
		),
		Logger:             clog.NewCompositeLogger(),
		ConnectionResolver: NewSqlServerConnectionResolver(),
		Options:            cconf.NewEmptyConfigParams(),
		retries:            DefaultRetriesCount,
	}
	return c
}

// Configure component by passing configuration parameters.
//	Parameters:
//		- ctx context.Context
//		- config configuration parameters to be set.
func (c *SqlServerConnection) Configure(ctx context.Context, config *cconf.ConfigParams) {
	config = config.SetDefaults(c.defaultConfig)
	c.ConnectionResolver.Configure(ctx, config)
	c.Options = c.Options.Override(config.GetSection("options"))

	c.DatabaseName, _ = config.GetAsNullableString("connection.database")
}

// SetReferences references to dependent components.
//	Parameters:
//		- ctx context.Context
//		- references references to locate the component dependencies.
func (c *SqlServerConnection) SetReferences(ctx context.Context, references cref.IReferences) {
	c.Logger.SetReferences(ctx, references)
	c.ConnectionResolver.SetReferences(ctx, references)
}

// IsOpen checks if the component is opened.
//	Returns true if the component has been opened and false otherwise.
func (c *SqlServerConnection) IsOpen() bool {
	return c.Connection != nil
}

//	Open the component.
//	Parameters:
//		- ctx context.Context
//		- correlationId 	(optional) transaction id to trace execution through call chain.
//		- Return 			error or nil no errors occurred.
func (c *SqlServerConnection) Open(ctx context.Context, correlationId string) error {

	uri, err := c.ConnectionResolver.Resolve(ctx, correlationId)
	if err != nil {
		c.Logger.Error(ctx, correlationId, err, "Failed to resolve SqlServer connection")
		return nil
	}

	c.Logger.Debug(ctx, correlationId, "Connecting to SqlServer")

	retries := c.retries
	for retries > 0 {
		pool, err := sql.Open("sqlserver", uri)
		if err != nil {
			retries--
			if retries <= 0 {
				return cerr.
					NewConnectionError(correlationId, "CONNECT_FAILED", "Connection to SqlServer failed").
					WithCause(err)
			}
			c.Logger.Debug(ctx, correlationId, "Failed to connect to SqlServers, try reconnect...")
			err = c.waitForRetry(ctx, correlationId, retries)
			if err != nil {
				return err
			}
			continue
		}
		idleTimeoutMS := c.Options.GetAsIntegerWithDefault("idle_timeout", DefaultIdleTimeout)
		maxPoolSize := c.Options.GetAsIntegerWithDefault("max_pool_size", DefaultMaxPoolSize)
		connectTimeoutMS := c.Options.GetAsIntegerWithDefault("connect_timeout", DefaultConnectTimeout)

		pool.SetConnMaxIdleTime(time.Duration(idleTimeoutMS) * time.Millisecond)
		pool.SetMaxOpenConns(maxPoolSize)
		pool.SetConnMaxLifetime(time.Duration(connectTimeoutMS) * time.Millisecond)

		c.Connection = pool
		break
	}
	return nil
}

// Close component and frees used resources.
//	Parameters:
//		- ctx context.Context
//		- correlationId (optional) transaction id to trace execution through call chain.
//	Returns: error or nil no errors occurred
func (c *SqlServerConnection) Close(ctx context.Context, correlationId string) error {
	if c.Connection == nil {
		return nil
	}
	c.Connection.Close()
	c.Logger.Debug(ctx, correlationId, "Disconnected from SqlServer database %s", c.DatabaseName)
	c.Connection = nil
	c.DatabaseName = ""
	return nil
}

func (c *SqlServerConnection) GetConnection() *sql.DB {
	return c.Connection
}

func (c *SqlServerConnection) GetDatabaseName() string {
	return c.DatabaseName
}

func (c *SqlServerConnection) waitForRetry(ctx context.Context, correlationId string, retries int) error {
	waitTime := DefaultConnectTimeout * int(math.Pow(float64(c.retries-retries), 2))

	select {
	case <-time.After(time.Duration(waitTime) * time.Millisecond):
		return nil
	case <-ctx.Done():
		return cerr.ApplicationErrorFactory.Create(
			&cerr.ErrorDescription{
				Type:          "Application",
				Category:      "Application",
				Code:          "CONTEXT_CANCELLED",
				Message:       "request canceled by parent context",
				CorrelationId: correlationId,
			},
		)
	}
}
