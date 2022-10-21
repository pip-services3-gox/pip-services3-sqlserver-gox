package build

import (
	cref "github.com/pip-services3-gox/pip-services3-commons-gox/refer"
	cbuild "github.com/pip-services3-gox/pip-services3-components-gox/build"
	conn "github.com/pip-services3-gox/pip-services3-sqlserver-gox/connect"
)

// DefaultSqlServerFactory creates SqlServer components by their descriptors.
//	see Factory
//	see SqlServerConnection
type DefaultSqlServerFactory struct {
	*cbuild.Factory
}

//	Create a new instance of the factory.
func NewDefaultSqlServerFactory() *DefaultSqlServerFactory {

	c := &DefaultSqlServerFactory{}
	c.Factory = cbuild.NewFactory()

	sqlserverConnectionDescriptor := cref.NewDescriptor("pip-services", "connection", "sqlserver", "*", "1.0")
	c.RegisterType(sqlserverConnectionDescriptor, conn.NewSqlServerConnection)

	return c
}
