package shared

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"zombiezen.com/go/sqlite"
)

func RunQuery(query string, output interface{}, params ...interface{}) error {
	conn, err := GetConn(context.Background())
	if err != nil {
		return err
	}
	defer PutConn(conn)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing query: %w", err)
	}
	defer stmt.Finalize()

	for i, param := range params {
		paramIndex := i + 1
		switch v := param.(type) {
		case string:
			stmt.BindText(paramIndex, v)
		case *string:
			if v == nil {
				stmt.BindNull(paramIndex)
			} else {
				stmt.BindText(paramIndex, *v)
			}
		case int64:
			stmt.BindInt64(paramIndex, v)
		case int:
			stmt.BindInt64(paramIndex, int64(v))
		case float64:
			stmt.BindFloat(paramIndex, v)
		case bool:
			stmt.BindBool(paramIndex, v)
		case []byte:
			stmt.BindBytes(paramIndex, v)
		case time.Time:
			stmt.BindText(paramIndex, v.UTC().Format(time.RFC3339))
		case *time.Time:
			if v == nil {
				stmt.BindNull(paramIndex)
			} else {
				stmt.BindText(paramIndex, v.UTC().Format(time.RFC3339))
			}
		case nil:
			stmt.BindNull(paramIndex)
		default:
			return fmt.Errorf("unsupported parameter type %T for parameter %d", param, paramIndex)
		}
	}

	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return fmt.Errorf("error executing query: %w", err)
		}
		if !hasRow {
			break
		}

		if output == nil {
			continue
		}

		outputValue := reflect.ValueOf(output)
		if outputValue.Kind() != reflect.Ptr {
			return fmt.Errorf("output must be a pointer")
		}

		elemValue := outputValue.Elem()

		switch elemValue.Kind() {
		case reflect.Int64, reflect.Int:
			elemValue.SetInt(stmt.ColumnInt64(0))

		case reflect.String:
			elemValue.SetString(stmt.ColumnText(0))

		case reflect.Struct:
			if err := scanStruct(stmt, elemValue); err != nil {
				return err
			}

		case reflect.Slice:
			elementType := elemValue.Type().Elem()
			newElement := reflect.New(elementType).Elem()
			if err := scanStruct(stmt, newElement); err != nil {
				return err
			}
			elemValue.Set(reflect.Append(elemValue, newElement))

		default:
			return fmt.Errorf("output must be a pointer to a struct, slice, or basic type")
		}
	}

	return nil
}

func scanStruct(stmt *sqlite.Stmt, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		switch field.Kind() {
		case reflect.Int64, reflect.Int:
			field.SetInt(stmt.ColumnInt64(i))

		case reflect.String:
			field.SetString(stmt.ColumnText(i))

		case reflect.Float64:
			field.SetFloat(stmt.ColumnFloat(i))

		case reflect.Bool:
			field.SetBool(stmt.ColumnBool(i))

		case reflect.Struct:
			if fieldType.Type == reflect.TypeOf(time.Time{}) {
				parsed, err := time.Parse(time.RFC3339, stmt.ColumnText(i))
				if err != nil {
					return fmt.Errorf("error parsing time for field %s: %w", fieldType.Name, err)
				}
				field.Set(reflect.ValueOf(parsed))
			} else {
				return fmt.Errorf("unsupported struct type for field %s", fieldType.Name)
			}

		case reflect.Ptr:
			if stmt.ColumnType(i) == sqlite.TypeNull {
				// leave as nil
			} else if fieldType.Type == reflect.TypeOf((*string)(nil)) {
				s := stmt.ColumnText(i)
				field.Set(reflect.ValueOf(&s))
			} else if fieldType.Type == reflect.TypeOf((*time.Time)(nil)) {
				parsed, err := time.Parse(time.RFC3339, stmt.ColumnText(i))
				if err != nil {
					return fmt.Errorf("error parsing time for field %s: %w", fieldType.Name, err)
				}
				field.Set(reflect.ValueOf(&parsed))
			} else {
				return fmt.Errorf("unsupported pointer type for field %s", fieldType.Name)
			}

		default:
			return fmt.Errorf("unsupported field type %v for field %s", field.Kind(), fieldType.Name)
		}
	}
	return nil
}
