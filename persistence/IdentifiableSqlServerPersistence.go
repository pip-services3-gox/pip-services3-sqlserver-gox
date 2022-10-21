package persistence

import (
	"context"
	"strconv"

	cerr "github.com/pip-services3-gox/pip-services3-commons-gox/errors"
	cpersist "github.com/pip-services3-gox/pip-services3-data-gox/persistence"

	cdata "github.com/pip-services3-gox/pip-services3-commons-gox/data"
)

// IdentifiableSqlServerPersistence Abstract persistence component that stores data in SqlServer
// and implements a number of CRUD operations over data items with unique ids.
// The data items must implement IIdentifiable interface.
//
// In basic scenarios child classes shall only override GetPageByFilter,
// getListByFilter or deleteByFilter operations with specific filter function.
// All other operations can be used out of the box.
//
// In complex scenarios child classes can implement additional operations by
// accessing c._collection and c._model properties.
//
//	Configuration parameters
//		- collection:               (optional) SqlServer collection name
//		- connection(s):
//			- discovery_key:        (optional) a key to retrieve the connection from IDiscovery
//			- host:                 host name or IP address
//			- port:                 port number (default: 27017)
//			- uri:                  resource URI or connection string with all parameters in it
//		- credential(s):
//			- store_key:            (optional) a key to retrieve the credentials from ICredentialStore
//			- username:             (optional) user name
//			- password:             (optional) user password
//		- options:
//			- connect_timeout:      (optional) number of milliseconds to wait before timing out when connecting a new client (default: 0)
//			- idle_timeout:         (optional) number of milliseconds a client must sit idle in the pool and not be checked out (default: 10000)
//			- max_pool_size:        (optional) maximum number of clients the pool should contain (default: 10)
//
//	References
//		- *:logger:*:*:1.0           (optional) ILogger components to pass log messages components to pass log messages
//		- *:discovery:*:*:1.0        (optional) IDiscovery services
//		- *:credential-store:*:*:1.0 (optional) Credential stores to resolve credentials
//
// Example:
//	type SqlServerServerPersistence struct {
//		*persist.IdentifiableSqlServerPersistence[MyData, string]
//	}
//
//	func NewSqlServerServerPersistence() *SqlServerServerPersistence {
//		c := &SqlServerServerPersistence{}
//		c.IdentifiableSqlServerPersistence = persist.InheritIdentifiableSqlServerPersistence[MyData, string](c, "mydata")
//		return c
//	}
//
//	func (c *SqlServerServerPersistence) DefineSchema() {
//		c.ClearSchema()
//		c.EnsureSchema("CREATE TABLE [" + c.TableName + "] ([id] VARCHAR(32) PRIMARY KEY, [key] VARCHAR(50), [content] VARCHAR(MAX))")
//		c.EnsureIndex(c.IdentifiableSqlServerPersistence.TableName+"_key", map[string]string{"key": "1"}, map[string]string{"unique": "true"})
//	}
//
//	func (c *SqlServerServerPersistence) GetPageByFilter(ctx context.Context, correlationId string,
//		filter cdata.FilterParams, paging cdata.PagingParams) (page cdata.DataPage[MyData], err error) {
//
//		key, ok := filter.GetAsNullableString("Key")
//		filterObj := ""
//		if ok && key != "" {
//			filterObj += "[key]='" + key + "'"
//		}
//		sorting := ""
//
//		return c.IdentifiableSqlServerPersistence.GetPageByFilter(ctx, correlationId,
//			filterObj, paging,
//			sorting, "",
//		)
//	}
//
//	func main() {
//		persistence := NewSqlServerServerPersistence()
//
//		persistence.Configure(context.Background(), NewConfigParamsFromTuples(
//			"host", "localhost",
//			"port", 1433,
//		))
//
//		err := persistence.Open(context.Background(), "123")
//
//		item, err := persistence.Create(context.Background(), "123", MyData{Id: "1", Name: "ABC"})
//		page, err := persistence.GetPageByFilter(context.Background(), "123", *NewFilterParamsFromTuples("name", "ABC"), nil)
//
//		fmt.Println(page.Data)
//		res, err := persistence.DeleteById(context.Background(), "123", "1")
//	}
//
type IdentifiableSqlServerPersistence[T any, K any] struct {
	*SqlServerPersistence[T]
}

// InheritIdentifiableSqlServerPersistence creates a new instance of the persistence component.
//	Parameters:
//		- ctx context.Context
//		- overrides References to override virtual methods
//		- tableName    (optional) a table name.
func InheritIdentifiableSqlServerPersistence[T any, K any](overrides ISqlServerPersistenceOverrides[T], tableName string) *IdentifiableSqlServerPersistence[T, K] {
	if tableName == "" {
		panic("Table name could not be empty")
	}

	c := &IdentifiableSqlServerPersistence[T, K]{}
	c.SqlServerPersistence = InheritSqlServerPersistence(overrides, tableName)

	return c
}

// GetListByIds gets a list of data items retrieved by given unique ids.
//	Parameters:
//		- ctx context.Context
//		- correlationId     (optional) transaction id to trace execution through call chain.
//		- ids of data items to be retrieved
//	Returns: a data list or error.
func (c *IdentifiableSqlServerPersistence[T, K]) GetListByIds(ctx context.Context, correlationId string,
	ids []K) (items []T, err error) {

	ln := len(ids)
	params := c.GenerateParameters(ln)
	query := "SELECT * FROM " + c.QuotedTableName() + " WHERE [id] IN(" + params + ")"

	rows, err := c.Client.QueryContext(ctx, query, ItemsToAnySlice(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items = make([]T, 0)
	for rows.Next() {
		if c.IsTerminated() {
			rows.Close()
			return nil, cerr.
				NewError("query terminated").
				WithCorrelationId(correlationId)
		}
		item, convErr := c.Overrides.ConvertToPublic(rows)
		if convErr != nil {
			return items, convErr
		}
		items = append(items, item)
	}

	if items != nil {
		c.Logger.Trace(ctx, correlationId, "Retrieved %d from %s", len(items), c.TableName)
	}

	return items, rows.Err()
}

// GetOneById gets a data item by its unique id.
//	Parameters:
//		- ctx context.Context
//		- correlationId     (optional) transaction id to trace execution through call chain.
//		- id                an id of data item to be retrieved.
// Returns: data item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) GetOneById(ctx context.Context, correlationId string, id K) (item T, err error) {

	query := "SELECT * FROM " + c.QuotedTableName() + " WHERE [id]=@p1"

	rows, err := c.Client.QueryContext(ctx, query, id)
	if err != nil {
		return item, err
	}
	defer rows.Close()

	if !rows.Next() {
		return item, rows.Err()
	}

	if err == nil {
		c.Logger.Trace(ctx, correlationId, "Retrieved from %s with id = %s", c.TableName, id)
		return c.Overrides.ConvertToPublic(rows)
	}
	c.Logger.Trace(ctx, correlationId, "Nothing found from %s with id = %s", c.TableName, id)
	return item, err
}

// Create a data item.
//	Parameters:
//		- ctx context.Context
//		- correlation_id    (optional) transaction id to trace execution through call chain.
//		- item              an item to be created.
//	Returns: (optional)  created item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) Create(ctx context.Context, correlationId string, item T) (result T, err error) {
	newItem := c.cloneItem(item)
	newItem = GenerateObjectIdIfNotExists[T](newItem)

	return c.SqlServerPersistence.Create(ctx, correlationId, newItem)
}

// Set a data item. If the data item exists it updates it,
// otherwise it creates a new data item.
//	Parameters:
//		- ctx context.Context
//		- correlation_id    (optional) transaction id to trace execution through call chain.
//		- item              an item to be set.
//	Returns: (optional)  updated item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) Set(ctx context.Context, correlationId string, item T) (result T, err error) {
	objMap, convErr := c.Overrides.ConvertFromPublic(item)
	if convErr != nil {
		return result, convErr
	}

	GenerateObjectMapIdIfNotExists(objMap)

	columns, values := c.GenerateColumnsAndValues(objMap)

	paramsStr := c.GenerateParameters(len(values))
	columnsStr := c.GenerateColumns(columns)
	setParams := c.GenerateSetParameters(columns)
	id := cpersist.GetObjectId(objMap)

	query := "INSERT INTO " + c.QuotedTableName() + " (" + columnsStr + ") OUTPUT INSERTED.* VALUES (" + paramsStr + ")"

	rows, err := c.Client.QueryContext(ctx, query, values...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if rows.Next() {
		result, convErr = c.Overrides.ConvertToPublic(rows)
		if convErr != nil {
			return result, convErr
		}
		c.Logger.Trace(ctx, correlationId, "Set in %s with id = %s", c.TableName, id)
		return result, nil
	}

	values = append(values, id)
	query = "UPDATE " + c.QuotedTableName() + " SET " + setParams + " OUTPUT INSERTED.* WHERE [id]=@p" + strconv.FormatInt(int64(len(values)), 10)
	rows, err = c.Client.QueryContext(ctx, query, values...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if !rows.Next() {
		return result, rows.Err()
	}

	result, convErr = c.Overrides.ConvertToPublic(rows)
	if convErr != nil {
		return result, convErr
	}
	c.Logger.Trace(ctx, correlationId, "Set in %s with id = %s", c.TableName, id)

	return result, rows.Err()

}

// Update a data item.
//	Parameters:
//		- ctx context.Context
//		- correlation_id    (optional) transaction id to trace execution through call chain.
//		- item              an item to be updated.
//	Returns          (optional)  updated item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) Update(ctx context.Context, correlationId string, item T) (result T, err error) {
	objMap, convErr := c.Overrides.ConvertFromPublic(item)
	if convErr != nil {
		return result, convErr
	}
	columns, values := c.GenerateColumnsAndValues(objMap)
	paramsStr := c.GenerateSetParameters(columns)
	id := cpersist.GetObjectId(objMap)
	values = append(values, id)

	query := "UPDATE " + c.QuotedTableName() + " SET " + paramsStr + " OUTPUT INSERTED.* WHERE [id]=@p" + strconv.FormatInt(int64(len(values)), 10)

	rows, err := c.Client.QueryContext(ctx, query, values...)
	if err != nil {
		return result, err
	}

	defer rows.Close()
	if !rows.Next() {
		return result, rows.Err()
	}

	if err == nil {
		result, convErr = c.Overrides.ConvertToPublic(rows)
		if convErr != nil {
			return result, convErr
		}
		c.Logger.Trace(ctx, correlationId, "Updated in %s with id = %s", c.TableName, id)
		return result, nil
	}
	return result, err
}

// UpdatePartially updates only few selected fields in a data item.
//	Parameters:
//		- ctx context.Context
//		- correlation_id    (optional) transaction id to trace execution through call chain.
//		- id                an id of data item to be updated.
//		- data              a map with fields to be updated.
//	Returns: updated item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) UpdatePartially(ctx context.Context, correlationId string, id K, data cdata.AnyValueMap) (result T, err error) {
	objMap, convErr := c.Overrides.ConvertFromPublicPartial(data.Value())
	if convErr != nil {
		return result, convErr
	}
	columns, values := c.GenerateColumnsAndValues(objMap)
	paramsStr := c.GenerateSetParameters(columns)
	values = append(values, id)

	query := "UPDATE " + c.QuotedTableName() + " SET " + paramsStr + " OUTPUT INSERTED.* WHERE [id]=@p" + strconv.FormatInt(int64(len(values)), 10)

	rows, err := c.Client.QueryContext(ctx, query, values...)
	if err != nil {
		return result, err
	}

	defer rows.Close()

	if !rows.Next() {
		return result, rows.Err()
	}

	if err == nil {
		result, convErr = c.Overrides.ConvertToPublic(rows)
		if convErr != nil {
			return result, convErr
		}
		c.Logger.Trace(ctx, correlationId, "Updated partially in %s with id = %s", c.TableName, id)
		return result, nil
	}
	return result, rows.Err()
}

// DeleteById deletes a data item by its unique id.
//	Parameters:
//		- ctx context.Context
//		- correlation_id    (optional) transaction id to trace execution through call chain.
//		- id                an id of the item to be deleted
//	Returns: (optional)  deleted item or error.
func (c *IdentifiableSqlServerPersistence[T, K]) DeleteById(ctx context.Context, correlationId string, id K) (result T, err error) {
	query := "DELETE FROM " + c.QuotedTableName() + " OUTPUT DELETED.* WHERE [id]=@p1"

	rows, err := c.Client.QueryContext(ctx, query, []any{id}...)
	if err != nil {
		return result, err
	}

	defer rows.Close()

	if !rows.Next() {
		return result, rows.Err()
	}

	if err == nil {
		result, convErr := c.Overrides.ConvertToPublic(rows)
		if convErr != nil {
			return result, convErr
		}
		c.Logger.Trace(ctx, correlationId, "Deleted from %s with id = %s", c.TableName, id)
		return result, nil
	}
	return result, rows.Err()
}

// DeleteByIds deletes multiple data items by their unique ids.
//	Parameters:
//		- ctx context.Context
//		- correlationId     (optional) transaction id to trace execution through call chain.
//		- ids                of data items to be deleted.
//	Returns: (optional)  error or null for success.
func (c *IdentifiableSqlServerPersistence[T, K]) DeleteByIds(ctx context.Context, correlationId string, ids []K) error {

	ln := len(ids)
	paramsStr := c.GenerateParameters(ln)

	query := "DELETE FROM " + c.QuotedTableName() + " WHERE \"id\" IN(" + paramsStr + ")"

	result, err := c.Client.ExecContext(ctx, query, ItemsToAnySlice(ids)...)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count != 0 {
		c.Logger.Trace(ctx, correlationId, "Deleted %d items from %s", count, c.TableName)
	}
	return nil
}
