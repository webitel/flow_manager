package flow

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/webitel/flow_manager/cases"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type GetCasesArgs struct {
	Contact struct {
		Id int64
	}
	Status struct {
		Id int64
	}
	Filters struct {
		DateFrom string
		DateTo   string
	}
	Limit  int64
	Offset int64
	Token  string
	SetVar string
}

// Define the fields to be returned in the search results
var fields = []string{"id", "subject", "author", "status", "created_at"}

func (r *router) getCases(
	ctx context.Context,
	scope *Flow,
	conn model.Connection,
	args any,
) (model.Response, *model.AppError) {
	// * set default limit to 100
	var argv GetCasesArgs = GetCasesArgs{Limit: 100}
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Token == "" {
		conn.Log().With(
			wlog.Int("schema_id", scope.schemaId),
			wlog.String("schema_name", scope.name),
		).Error("Token is required")

		return nil, model.NewAppError("getCases", "missing_token", nil, "Token is required", 400)
	}

	if argv.SetVar == "" {
		conn.Log().With(
			wlog.Int("schema_id", scope.schemaId),
			wlog.String("schema_name", scope.name),
		).Error("SetVar is required")

		return nil, model.NewAppError("getCases", "missing_set_var", nil, "SetVar is required", 400)
	}

	if argv.Contact.Id == 0 {
		conn.Log().With(
			wlog.Int("schema_id", scope.schemaId),
			wlog.String("schema_name", scope.name),
		).Error("Contact.Id is required")

		return nil, model.NewAppError("getCases", "missing_contact_id", nil, "Contact.Id is required", 400)
	}

	// Build Filters map dynamically with non-empty values
	filters := make(map[string]string)

	// Add filter for author if contact ID is provided
	if argv.Contact.Id != 0 {
		ID := strconv.Itoa(int(argv.Contact.Id))
		filters["author"] = ID
	}

	// Add status filter if provided
	if argv.Status.Id != 0 {
		ID := strconv.Itoa(int(argv.Status.Id))
		filters["status"] = ID
	}

	// Add date range filters if provided
	if argv.Filters.DateFrom != "" {
		filters["created_at.from"] = argv.Filters.DateFrom
	}
	if argv.Filters.DateTo != "" {
		filters["created_at.to"] = argv.Filters.DateTo
	}

	// * Perform the search with the dynamically built filters
	res, err := r.fm.SearchCases(ctx, &cases.SearchCasesRequest{
		Token:   argv.Token,
		Limit:   argv.Limit,
		Offset:  argv.Offset,
		Fields:  fields,
		Filters: filters,
	})
	if err != nil {
		conn.Log().With(
			wlog.Int("schema_id", scope.schemaId),
			wlog.String("schema_name", scope.name),
		).Error(err.Error())

		return nil, model.NewAppError("getCases", "get_cases_failed", nil, err.Error(), 500)
	}

	// Marshal the response into JSON
	casesJSON, err := json.Marshal(res)
	if err != nil {
		conn.Log().With(
			wlog.Int("schema_id", scope.schemaId),
			wlog.String("schema_name", scope.name),
		).Error(err.Error())

		return nil, model.NewAppError("getCases", "json_encode_failed", nil, err.Error(), 500)
	}

	// Set the result in the response variables
	return conn.Set(ctx, model.Variables{
		argv.SetVar: string(casesJSON),
	})
}
