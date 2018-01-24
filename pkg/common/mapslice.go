package common

import yaml "gopkg.in/yaml.v2"

type mapSlice yaml.MapSlice

func (s mapSlice) get(key string) interface{} {
	for _, i := range s {
		k := i.Key.(string)
		if k == key {
			return i.Value
		}
	}
	return nil
}

func (s mapSlice) keys() []string {
	result := []string{}
	for _, i := range s {
		k := i.Key.(string)
		result = append(result, k)
	}
	return result
}

func (s mapSlice) getString(key string) string {
	i := s.get(key)
	result, _ := i.(string)
	return result
}

func (s mapSlice) getInt(key string) int {
	i := s.get(key)
	result, _ := i.(int)
	return result
}

func (s mapSlice) getMapSlice(key string) mapSlice {
	i := s.get(key)
	if i == nil {
		return nil
	}
	result := i.(yaml.MapSlice)
	return mapSlice(result)
}

func (s mapSlice) getStringSlice(key string) []string {
	i := s.get(key)
	if i == nil {
		return nil
	}
	iResult := i.([]interface{})
	result := []string{}
	for _, a := range iResult {
		result = append(result, a.(string))
	}
	return result
}

//Copy copies a json schema
func (s mapSlice) JSONSchema() *JSONSchema {
	if s == nil {
		return nil
	}
	properties := s.getMapSlice("properties")
	schema := &JSONSchema{
		Title:           s.getString("title"),
		SQL:             s.getString("sql"),
		Default:         s.get("default"),
		Enum:            s.getStringSlice("enum"),
		Minimum:         s.getInt("minimum"),
		Maximum:         s.getInt("maximum"),
		Ref:             s.getString("$ref"),
		Permission:      s.getStringSlice("permission"),
		Operation:       s.getString("operation"),
		Type:            s.getString("type"),
		Presence:        s.getString("presence"),
		Properties:      map[string]*JSONSchema{},
		PropertiesOrder: properties.keys(),
	}
	if properties == nil {
		schema.Properties = nil
	}
	for _, property := range properties {
		key := property.Key.(string)
		schema.Properties[key] = mapSlice(property.Value.(yaml.MapSlice)).JSONSchema()
	}
	items := s.getMapSlice("items")
	schema.Items = items.JSONSchema()
	return schema
}