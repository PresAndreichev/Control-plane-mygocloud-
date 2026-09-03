package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/olekukonko/tablewriter"
	"go.yaml.in/yaml/v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

func Print(data interface{}, format Format, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}

	switch format {
	case FormatJSON:
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		return yaml.NewEncoder(out).Encode(data)
	default:
		return printTable(data, out)
	}
}

func printTable(data interface{}, out io.Writer) error {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		return printStructTable(v, out)
	}

	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			fmt.Fprintln(out, "No data.")
			return nil
		}
		return printSliceTable(v, out)
	}
	fmt.Fprintf(out, "%+v\n", data)
	return nil
}

func printStructTable(v reflect.Value, out io.Writer) error {
	t := tablewriter.NewWriter(out)
	t.Header([]string{"Field", "Value"})

	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name := jsonTagName(field)
		val := fmt.Sprintf("%v", v.Field(i).Interface())
		t.Append([]string{name, val})
	}

	t.Render()
	return nil
}

func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	parts := strings.Split(tag, ",")
	return parts[0]
}

func printSliceTable(v reflect.Value, out io.Writer) error {
	if v.Len() == 0 {
		return nil
	}

	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	if elem.Kind() != reflect.Struct {
		fmt.Fprintf(out, "%+v\n", v.Interface())
		return nil
	}

	t := tablewriter.NewWriter(out)
	typ := elem.Type()

	var headers []string
	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		headers = append(headers, jsonTagName(field))
	}
	t.Header(headers)

	for i := 0; i < v.Len(); i++ {
		rowVal := v.Index(i)
		if rowVal.Kind() == reflect.Ptr {
			rowVal = rowVal.Elem()
		}
		var row []string
		for j := 0; j < rowVal.NumField(); j++ {
			field := typ.Field(j)
			if !field.IsExported() {
				continue
			}
			val := fmt.Sprintf("%v", rowVal.Field(j).Interface())
			if len(val) > 60 {
				val = val[:57] + "..."
			}
			row = append(row, val)
		}
		t.Append(row)

	}
	t.Render()
	return nil

}
