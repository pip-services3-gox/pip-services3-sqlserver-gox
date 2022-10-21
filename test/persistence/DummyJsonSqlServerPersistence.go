package test

import (
	"context"

	cdata "github.com/pip-services3-gox/pip-services3-commons-gox/data"
	persist "github.com/pip-services3-gox/pip-services3-sqlserver-gox/persistence"
	"github.com/pip-services3-gox/pip-services3-sqlserver-gox/test/fixtures"
)

type DummyJsonSqlServerPersistence struct {
	*persist.IdentifiableJsonSqlServerPersistence[fixtures.Dummy, string]
}

func NewDummyJsonSqlServerPersistence() *DummyJsonSqlServerPersistence {
	c := &DummyJsonSqlServerPersistence{}
	c.IdentifiableJsonSqlServerPersistence = persist.InheritIdentifiableJsonSqlServerPersistence[fixtures.Dummy, string](c, "dummies_json")
	return c
}

func (c *DummyJsonSqlServerPersistence) DefineSchema() {
	c.ClearSchema()
	c.EnsureTable("", "")
	c.EnsureSchema("ALTER TABLE [" + c.TableName + "] ADD [data_key] AS JSON_VALUE([data],'$.key')")
	c.EnsureIndex(c.TableName+"_key", map[string]string{"data_key": "1"}, map[string]string{"unique": "true"})
}

func (c *DummyJsonSqlServerPersistence) GetPageByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams, paging cdata.PagingParams) (page cdata.DataPage[fixtures.Dummy], err error) {

	key, ok := filter.GetAsNullableString("Key")
	filterObj := ""
	if ok && key != "" {
		filterObj += "JSON_VALUE([data],'$.key')='" + key + "'"
	}

	return c.IdentifiableJsonSqlServerPersistence.GetPageByFilter(ctx, correlationId,
		filterObj, paging,
		"", "",
	)
}

func (c *DummyJsonSqlServerPersistence) GetCountByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams) (count int64, err error) {

	filterObj := ""
	if key, ok := filter.GetAsNullableString("Key"); ok && key != "" {
		filterObj += "JSON_VALUE([data],'$.key')='" + key + "'"
	}

	return c.IdentifiableJsonSqlServerPersistence.GetCountByFilter(ctx, correlationId, filterObj)
}

func (c *DummyJsonSqlServerPersistence) GetOneRandom(ctx context.Context, correlationId string) (item fixtures.Dummy, err error) {
	return c.IdentifiableJsonSqlServerPersistence.GetOneRandom(ctx, correlationId, "")
}
