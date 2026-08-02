package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	"baniru-test/internal/models"
	"baniru-test/internal/store"
)

func NewServer(st *store.Store) http.Handler {
	schema, err := buildSchema(st)
	if err != nil {
		panic(err)
	}

	h := handler.New(&handler.Config{
		Schema: &schema,
		Pretty: true,
	})

	mux := http.NewServeMux()
	mux.Handle("/graphql", h)
	return mux
}

func buildSchema(st *store.Store) (graphql.Schema, error) {
	subObjEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "SubObj",
		Values: graphql.EnumValueConfigMap{
			"tools":  &graphql.EnumValueConfig{Value: string(models.SubObjTools)},
			"tables": &graphql.EnumValueConfig{Value: string(models.SubObjTables)},
			"chairs": &graphql.EnumValueConfig{Value: string(models.SubObjChairs)},
		},
	})

	chairTypeEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "ChairType",
		Values: graphql.EnumValueConfigMap{
			"abc": &graphql.EnumValueConfig{Value: string(models.ChairTypeABC)},
			"cde": &graphql.EnumValueConfig{Value: string(models.ChairTypeCDE)},
		},
	})

	mainType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Main",
		Fields: graphql.Fields{
			"id":           &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subId":        &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"subObj":       &graphql.Field{Type: graphql.NewNonNull(subObjEnum)},
			"createdAt":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"updateAt":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description1": &graphql.Field{Type: graphql.String},
			"description2": &graphql.Field{Type: graphql.String},
			"description3": &graphql.Field{Type: graphql.String},
			"type":         &graphql.Field{Type: chairTypeEnum},
		},
	})

	mainInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "MainInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"title":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"subObj":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(subObjEnum)},
			"description1": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"description2": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"description3": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"type":         &graphql.InputObjectFieldConfig{Type: chairTypeEnum},
		},
	})

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"main": &graphql.Field{
				Type: mainType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					m, err := st.GetMain(p.Context, id)
					if err != nil {
						return nil, err
					}
					if m == nil {
						return nil, nil
					}
					return toMap(m), nil
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createMain": &graphql.Field{
				Type: graphql.NewNonNull(mainType),
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(mainInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					in, err := parseInput(p.Args["input"])
					if err != nil {
						return nil, err
					}
					m, err := st.CreateMain(p.Context, in)
					if err != nil {
						return nil, err
					}
					return toMap(m), nil
				},
			},
			"updateMain": &graphql.Field{
				Type: graphql.NewNonNull(mainType),
				Args: graphql.FieldConfigArgument{
					"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(mainInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					in, err := parseInput(p.Args["input"])
					if err != nil {
						return nil, err
					}
					m, err := st.UpdateMain(p.Context, id, in)
					if err != nil {
						return nil, err
					}
					return toMap(m), nil
				},
			},
			"deleteMain": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					return st.SoftDeleteMain(p.Context, id)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
}

func parseID(v interface{}) (int, error) {
	switch t := v.(type) {
	case string:
		return strconv.Atoi(t)
	case int:
		return t, nil
	case int32:
		return int(t), nil
	case int64:
		return int(t), nil
	case json.Number:
		i, err := t.Int64()
		return int(i), err
	default:
		return 0, fmt.Errorf("некорректный id: %v", v)
	}
}

func parseInput(v interface{}) (models.MainInput, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return models.MainInput{}, fmt.Errorf("некорректный input")
	}

	in := models.MainInput{
		Title:  strOrEmpty(m["title"]),
		SubObj: models.SubObj(strOrEmpty(m["subObj"])),
	}
	if d, ok := m["description1"].(string); ok {
		in.Description1 = &d
	}
	if d, ok := m["description2"].(string); ok {
		in.Description2 = &d
	}
	if d, ok := m["description3"].(string); ok {
		in.Description3 = &d
	}
	if t, ok := m["type"].(string); ok {
		ct := models.ChairType(t)
		in.Type = &ct
	}
	return in, nil
}

func strOrEmpty(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toMap(m *models.Main) map[string]interface{} {
	out := map[string]interface{}{
		"id":        strconv.Itoa(m.ID),
		"title":     m.Title,
		"subId":     m.SubID,
		"subObj":    string(m.SubObj),
		"createdAt": m.CreatedAt.UTC().Format(time.RFC3339),
		"updateAt":  m.UpdateAt.UTC().Format(time.RFC3339),
	}
	if m.Description1 != nil {
		out["description1"] = *m.Description1
	}
	if m.Description2 != nil {
		out["description2"] = *m.Description2
	}
	if m.Description3 != nil {
		out["description3"] = *m.Description3
	}
	if m.Type != nil {
		out["type"] = string(*m.Type)
	}
	return out
}
