package test

import (
	"context"

	cdata "github.com/pip-services3-gox/pip-services3-commons-gox/data"
	persist "github.com/pip-services3-gox/pip-services3-sqlserver-gox/persistence"
)

type DummyMapSqlServerPersistence struct {
	*persist.IdentifiableSqlServerPersistence[map[string]any, string]
}

func NewDummyMapSqlServerPersistence() *DummyMapSqlServerPersistence {
	c := &DummyMapSqlServerPersistence{}
	c.IdentifiableSqlServerPersistence = persist.InheritIdentifiableSqlServerPersistence[map[string]any, string](c, "dummies")
	return c
}

func (c *DummyMapSqlServerPersistence) DefineSchema() {
	c.ClearSchema()
	c.EnsureSchema("CREATE TABLE [" + c.TableName + "] ([id] VARCHAR(32) PRIMARY KEY, [key] VARCHAR(50), [content] VARCHAR(MAX))")
	c.EnsureIndex(c.IdentifiableSqlServerPersistence.TableName+"_key", map[string]string{"key": "1"}, map[string]string{"unique": "true"})
}

func (c *DummyMapSqlServerPersistence) GetPageByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams, paging cdata.PagingParams) (page cdata.DataPage[map[string]any], err error) {

	key, ok := filter.GetAsNullableString("Key")
	filterObj := ""
	if ok && key != "" {
		filterObj += "[key]='" + key + "'"
	}
	sorting := ""

	return c.IdentifiableSqlServerPersistence.GetPageByFilter(ctx, correlationId,
		filterObj, paging, sorting, "",
	)
}

func (c *DummyMapSqlServerPersistence) GetCountByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams) (count int64, err error) {

	key, ok := filter.GetAsNullableString("Key")
	filterObj := ""
	if ok && key != "" {
		filterObj += "[key]='" + key + "'"
	}
	return c.IdentifiableSqlServerPersistence.GetCountByFilter(ctx, correlationId, filterObj)
}
