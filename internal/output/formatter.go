// Package output provides formatters for displaying GitLab data in various formats.
package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var ErrUnsupportedFormat = errors.New("unsupported format")

type Format string

const (
	// FormatTable represents table output format.
	FormatTable Format = "table"
	// FormatJSON represents JSON output format.
	FormatJSON Format = "json"
	// FormatCSV represents CSV output format.
	FormatCSV Format = "csv"

	defaultExpiresAtText string = "Never"
	defaultLastUsedText  string = "Never"
)

type Formatter interface {
	FormatGroups(groups []*gitlab.Group) error
	FormatProjects(projects []*gitlab.Project) error
	FormatGroupAccessTokens(tokens []*glclient.GroupAccessTokenWithGroup) error
	FormatProjectAccessTokens(tokens []*glclient.ProjectAccessTokenWithProject) error
	FormatPipelineTriggers(triggers []*glclient.PipelineTriggerWithProject) error
	FormatProjectVariables(variables []*glclient.ProjectVariableWithProject) error
}

func NewFormatter(format Format) (Formatter, error) {
	switch format {
	case FormatTable:
		return &TableFormatter{}, nil
	case FormatJSON:
		return &JSONFormatter{}, nil
	case FormatCSV:
		return &CSVFormatter{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

type TableFormatter struct{}

func (f *TableFormatter) FormatGroups(groups []*gitlab.Group) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Full Path"})

	for _, group := range groups {
		fullPathLink := text.Hyperlink(group.WebURL, group.FullPath)
		t.AppendRow(table.Row{group.ID, group.Name, fullPathLink})
	}

	t.Render()

	return nil
}

func (f *TableFormatter) FormatProjects(projects []*gitlab.Project) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Path with Namespace"})

	for _, project := range projects {
		pathLink := text.Hyperlink(project.WebURL, project.PathWithNamespace)
		t.AppendRow(table.Row{project.ID, project.Name, pathLink})
	}

	t.Render()

	return nil
}

func (f *TableFormatter) FormatGroupAccessTokens(tokens []*glclient.GroupAccessTokenWithGroup) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Group Path", "Token Name", "Scopes", "Active", "Expires At"})

	for _, token := range tokens {
		expiresAt := defaultExpiresAtText
		if token.ExpiresAt != nil {
			expiresAt = token.ExpiresAt.String()
		}

		groupURL := token.GroupWebURL + "/-/settings/access_tokens"
		groupPathLink := text.Hyperlink(groupURL, token.GroupPath)

		t.AppendRow(table.Row{groupPathLink, token.Name, token.Scopes, token.Active, expiresAt})
	}

	t.Render()

	return nil
}

func (f *TableFormatter) FormatProjectAccessTokens(tokens []*glclient.ProjectAccessTokenWithProject) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Project Path", "Token Name", "Scopes", "Active", "Expires At"})

	for _, token := range tokens {
		expiresAt := defaultExpiresAtText
		if token.ExpiresAt != nil {
			expiresAt = token.ExpiresAt.String()
		}

		projectURL := token.ProjectWebURL + "/-/settings/access_tokens"
		projectPathLink := text.Hyperlink(projectURL, token.ProjectPath)

		t.AppendRow(table.Row{projectPathLink, token.Name, token.Scopes, token.Active, expiresAt})
	}

	t.Render()

	return nil
}

func (f *TableFormatter) FormatPipelineTriggers(triggers []*glclient.PipelineTriggerWithProject) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Project Path", "Description", "Owner", "Created At", "Last Used"})

	for _, trigger := range triggers {
		owner := "N/A"
		if trigger.Owner != nil {
			owner = trigger.Owner.Username
		}

		lastUsed := defaultLastUsedText
		if trigger.LastUsed != nil {
			lastUsed = trigger.LastUsed.String()
		}

		createdAt := "N/A"
		if trigger.CreatedAt != nil {
			createdAt = trigger.CreatedAt.String()
		}

		projectURL := trigger.ProjectWebURL + "/-/settings/ci_cd#js-pipeline-triggers"
		projectPathLink := text.Hyperlink(projectURL, trigger.ProjectPath)

		t.AppendRow(table.Row{projectPathLink, trigger.Description, owner, createdAt, lastUsed})
	}

	t.Render()

	return nil
}

func (f *TableFormatter) FormatProjectVariables(variables []*glclient.ProjectVariableWithProject) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Project Path", "Key", "Type", "Protected", "Masked", "Environment"})

	for _, variable := range variables {
		projectURL := variable.ProjectWebURL + "/-/settings/ci_cd#js-cicd-variables-settings"
		projectPathLink := text.Hyperlink(projectURL, variable.ProjectPath)

		t.AppendRow(table.Row{
			projectPathLink,
			variable.Key,
			variable.VariableType,
			variable.Protected,
			variable.Masked,
			variable.EnvironmentScope,
		})
	}

	t.Render()

	return nil
}

type JSONFormatter struct{}

func (f *JSONFormatter) FormatGroups(groups []*gitlab.Group) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(groups); err != nil {
		return fmt.Errorf("failed to encode groups as JSON: %w", err)
	}

	return nil
}

func (f *JSONFormatter) FormatProjects(projects []*gitlab.Project) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(projects); err != nil {
		return fmt.Errorf("failed to encode projects as JSON: %w", err)
	}

	return nil
}

func (f *JSONFormatter) FormatGroupAccessTokens(tokens []*glclient.GroupAccessTokenWithGroup) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(tokens); err != nil {
		return fmt.Errorf("failed to encode group access tokens as JSON: %w", err)
	}

	return nil
}

func (f *JSONFormatter) FormatProjectAccessTokens(tokens []*glclient.ProjectAccessTokenWithProject) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(tokens); err != nil {
		return fmt.Errorf("failed to encode project access tokens as JSON: %w", err)
	}

	return nil
}

func (f *JSONFormatter) FormatPipelineTriggers(triggers []*glclient.PipelineTriggerWithProject) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(triggers); err != nil {
		return fmt.Errorf("failed to encode pipeline triggers as JSON: %w", err)
	}

	return nil
}

func (f *JSONFormatter) FormatProjectVariables(variables []*glclient.ProjectVariableWithProject) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(variables); err != nil {
		return fmt.Errorf("failed to encode project variables as JSON: %w", err)
	}

	return nil
}

type CSVFormatter struct{}

func (f *CSVFormatter) FormatGroups(groups []*gitlab.Group) error {
	if len(groups) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(groups[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, group := range groups {
		row := getCSVRow(group)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (f *CSVFormatter) FormatProjects(projects []*gitlab.Project) error {
	if len(projects) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(projects[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, project := range projects {
		row := getCSVRow(project)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (f *CSVFormatter) FormatGroupAccessTokens(tokens []*glclient.GroupAccessTokenWithGroup) error {
	if len(tokens) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(tokens[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, token := range tokens {
		row := getCSVRow(token)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (f *CSVFormatter) FormatProjectAccessTokens(tokens []*glclient.ProjectAccessTokenWithProject) error {
	if len(tokens) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(tokens[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, token := range tokens {
		row := getCSVRow(token)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (f *CSVFormatter) FormatProjectVariables(variables []*glclient.ProjectVariableWithProject) error {
	if len(variables) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(variables[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, variable := range variables {
		row := getCSVRow(variable)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func getCSVHeaders(v interface{}) []string {
	var headers []string

	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)

		switch {
		case field.Anonymous && field.Type.Kind() == reflect.Ptr:
			// Create a zero value of the embedded type to extract headers
			embeddedType := field.Type.Elem()
			embeddedVal := reflect.New(embeddedType)
			embeddedHeaders := getCSVHeaders(embeddedVal.Interface())
			headers = append(headers, embeddedHeaders...)
		case field.Anonymous && field.Type.Kind() == reflect.Struct:
			// Handle non-pointer embedded structs
			embeddedVal := reflect.New(field.Type)
			embeddedHeaders := getCSVHeaders(embeddedVal.Interface())
			headers = append(headers, embeddedHeaders...)
		default:
			// Regular field
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				headers = append(headers, jsonTag)
			}
		}
	}

	return headers
}

func getCSVRow(v interface{}) []string {
	var row []string

	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		switch {
		case field.Anonymous && field.Type.Kind() == reflect.Ptr:
			row = append(row, getEmbeddedCSVRow(fieldValue)...)
		case field.Anonymous && field.Type.Kind() == reflect.Struct:
			row = append(row, getEmbeddedCSVRow(fieldValue.Addr())...)
		default:
			// Regular field
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				row = append(row, fmt.Sprintf("%v", fieldValue.Interface()))
			}
		}
	}

	return row
}

func getEmbeddedCSVRow(fieldValue reflect.Value) []string {
	var row []string

	if !fieldValue.IsNil() {
		embeddedRow := getCSVRow(fieldValue.Interface())
		row = append(row, embeddedRow...)
	} else {
		// Count the number of fields in the embedded struct to add empty values
		embeddedType := fieldValue.Type().Elem()
		embeddedVal := reflect.New(embeddedType)
		embeddedHeaders := getCSVHeaders(embeddedVal.Interface())

		for range embeddedHeaders {
			row = append(row, "")
		}
	}

	return row
}

func (f *CSVFormatter) FormatPipelineTriggers(triggers []*glclient.PipelineTriggerWithProject) error {
	if len(triggers) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := getCSVHeaders(triggers[0])
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, trigger := range triggers {
		row := getCSVRow(trigger)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
