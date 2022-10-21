package test

import (
	"context"

	cdata "github.com/pip-services3-gox/pip-services3-commons-gox/data"
	persist "github.com/pip-services3-gox/pip-services3-sqlserver-gox/persistence"
	"github.com/pip-services3-gox/pip-services3-sqlserver-gox/test/fixtures"
)

type DummyRefSqlServerPersistence struct {
	*persist.IdentifiableSqlServerPersistence[*fixtures.Dummy, string]
}

func NewDummyRefSqlServerPersistence() *DummyRefSqlServerPersistence {
	c := &DummyRefSqlServerPersistence{}
	c.IdentifiableSqlServerPersistence = persist.InheritIdentifiableSqlServerPersistence[*fixtures.Dummy, string](c, "dummies")
	return c
}

func (c *DummyRefSqlServerPersistence) GetPageByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams, paging cdata.PagingParams) (page cdata.DataPage[*fixtures.Dummy], err error) {

	key, ok := filter.GetAsNullableString("Key")
	filterObj := ""
	if ok && key != "" {
		filterObj += "[key]='" + key + "'"
	}
	sorting := ""

	return c.IdentifiableSqlServerPersistence.GetPageByFilter(ctx, correlationId,
		filterObj, paging,
		sorting, "",
	)
}

func (c *DummyRefSqlServerPersistence) GetCountByFilter(ctx context.Context, correlationId string,
	filter cdata.FilterParams) (count int64, err error) {

	key, ok := filter.GetAsNullableString("Key")
	filterObj := ""
	if ok && key != "" {
		filterObj += "[key]='" + key + "'"
	}
	return c.IdentifiableSqlServerPersistence.GetCountByFilter(ctx, correlationId, filterObj)
}
