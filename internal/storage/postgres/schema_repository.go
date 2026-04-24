package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SchemaRepository struct {
	db infraSql.Store
}

func NewSchemaRepository(db infraSql.Store) store.SchemaStore {
	return &SchemaRepository{db: db}
}

type schemaRow struct {
	Id         int             `db:"id"`
	DomainId   int64           `db:"domain_id"`
	UpdatedAt  int64           `db:"updated_at"`
	Type       string          `db:"type"`
	DomainName string          `db:"domain_name"`
	Name       string          `db:"name"`
	Schema     json.RawMessage `db:"schema"`
	Debug      bool            `db:"debug"`
}

const getSchemaSQL = `
SELECT s.id, s.domain_id, d.name AS domain_name, s.name,
       s.scheme AS schema, s.type, s.updated_at,
       coalesce(s.debug, false) AS debug
  FROM flow.acr_routing_scheme s
  JOIN directory.wbt_domain d ON d.dc = s.domain_id
 WHERE s.domain_id = @domain_id AND s.id = @id`

func (r *SchemaRepository) Get(domainId int64, id int) (*model.Schema, error) {
	var row schemaRow
	if err := r.db.Get(context.Background(), &row, getSchemaSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"id":        id,
	}); err != nil {
		return nil, err
	}
	return toSchema(row)
}

const getUpdatedAtSQL = `
SELECT s.updated_at
  FROM flow.acr_routing_scheme s
 WHERE s.id = @id AND s.domain_id = @domain_id`

func (r *SchemaRepository) GetUpdatedAt(domainId int64, id int) (int64, error) {
	var updatedAt int64
	if err := r.db.Get(context.Background(), &updatedAt, getUpdatedAtSQL, pgx.NamedArgs{
		"id":        id,
		"domain_id": domainId,
	}); err != nil {
		return 0, err
	}
	return updatedAt, nil
}

type routingRow struct {
	SourceId        int             `db:"source_id"`
	SourceName      string          `db:"source_name"`
	SourceData      string          `db:"source_data"`
	DomainId        int64           `db:"domain_id"`
	DomainName      string          `db:"domain_name"`
	TimezoneId      int64           `db:"timezone_id"`
	TimezoneName    string          `db:"timezone_name"`
	SchemaId        int             `db:"scheme_id"`
	SchemaName      string          `db:"scheme_name"`
	SchemaUpdatedAt int64           `db:"schema_updated_at"`
	Debug           bool            `db:"debug"`
	Variables       json.RawMessage `db:"variables"`
}

const getTransferredRoutingSQL = `
SELECT sg.id AS source_id, sg.name AS source_name, 'transfer' AS source_data,
       d.dc AS domain_id, d.name AS domain_name,
       coalesce(d.timezone_id, 287) AS timezone_id,
       coalesce(ct.sys_name, 'UTC') AS timezone_name,
       sg.id AS scheme_id, sg.name AS scheme_name,
       sg.updated_at AS schema_updated_at,
       sg.debug,
       null::jsonb AS variables
  FROM flow.acr_routing_scheme sg
  LEFT JOIN directory.wbt_domain d ON sg.domain_id = d.dc
  LEFT JOIN flow.calendar_timezones ct ON d.timezone_id = ct.id
 WHERE sg.id = @schema_id AND sg.domain_id = @domain_id`

func (r *SchemaRepository) GetTransferredRouting(domainId int64, schemaId int) (*model.Routing, error) {
	var row routingRow
	if err := r.db.Get(context.Background(), &row, getTransferredRoutingSQL, pgx.NamedArgs{
		"schema_id": schemaId,
		"domain_id": domainId,
	}); err != nil {
		return nil, err
	}
	return toRouting(row), nil
}

type schemaVariableRow struct {
	Value   []byte `db:"value"`
	Encrypt bool   `db:"encrypt"`
}

const getVariableSQL = `
SELECT value #>> '{}' AS value, encrypt
  FROM flow.scheme_variable
 WHERE domain_id = @domain_id AND name = @name`

func (r *SchemaRepository) GetVariable(domainId int64, name string) (*model.SchemaVariable, error) {
	var row schemaVariableRow
	if err := r.db.Get(context.Background(), &row, getVariableSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"name":      name,
	}); err != nil {
		return nil, err
	}
	return &model.SchemaVariable{
		Value:   row.Value,
		Encrypt: row.Encrypt,
	}, nil
}

const setVariableSQL = `
INSERT INTO flow.scheme_variable (domain_id, name, value, encrypt)
VALUES (@domain_id, @name, @value, @encrypt)
ON CONFLICT (domain_id, name)
    DO UPDATE SET value = EXCLUDED.value, encrypt = EXCLUDED.encrypt`

func (r *SchemaRepository) SetVariable(domainId int64, name string, val *model.SchemaVariable) error {
	return r.db.Exec(context.Background(), setVariableSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"name":      name,
		"value":     val.Value,
		"encrypt":   val.Encrypt,
	})
}

func toSchema(row schemaRow) (*model.Schema, error) {
	var apps model.Applications
	if len(row.Schema) > 0 {
		if err := json.Unmarshal(row.Schema, &apps); err != nil {
			return nil, fmt.Errorf("schema: unmarshal applications: %w", err)
		}
	}
	return &model.Schema{
		Id:         row.Id,
		DomainId:   row.DomainId,
		UpdatedAt:  row.UpdatedAt,
		Type:       row.Type,
		DomainName: row.DomainName,
		Name:       row.Name,
		Schema:     apps,
		Debug:      row.Debug,
	}, nil
}

func toRouting(row routingRow) *model.Routing {
	return &model.Routing{
		SourceId:        row.SourceId,
		SourceName:      row.SourceName,
		SourceData:      row.SourceData,
		DomainId:        row.DomainId,
		DomainName:      row.DomainName,
		TimezoneId:      row.TimezoneId,
		TimezoneName:    row.TimezoneName,
		SchemaId:        row.SchemaId,
		SchemaName:      row.SchemaName,
		SchemaUpdatedAt: row.SchemaUpdatedAt,
		Debug:           row.Debug,
	}
}
