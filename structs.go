package main

import (
	"encoding/json"
	"time"
)

// Datasource struct used in listAgentTypes command
type Datasource struct {
	Items []struct {
		DatasetsURI string `json:"datasetsUri"`
		Label       string `json:"label"`
		Type        string `json:"type"`
		ID          string `json:"id"`
		URI         string `json:"uri"`
		Description string `json:"description"`
		ProviderID  string `json:"providerId"`
		WidgetsURI  string `json:"widgetsUri"`
		Version     string `json:"version,omitempty"`
	} `json:"items"`
	FilteredRows int    `json:"filteredRows"`
	Identifier   string `json:"identifier"`
	TotalRows    int    `json:"totalRows"`
	NumRows      int    `json:"numRows"`
}

// Dataset struct used in listAttributeGroup command
type Dataset struct {
	Items []struct {
		DatasourceID string `json:"datasourceId"`
		Label        string `json:"label"`
		Type         string `json:"type"`
		ID           string `json:"id"`
		ItemsURI     string `json:"itemsUri"`
		URI          string `json:"uri"`
		Description  string `json:"description"`
		ProviderID   string `json:"providerId"`
	} `json:"items"`
	FilteredRows int    `json:"filteredRows"`
	Identifier   string `json:"identifier"`
	TotalRows    int    `json:"totalRows"`
	NumRows      int    `json:"numRows"`
}

// Columns struct used in listAttributes command
type Columns struct {
	Items []struct {
		Affver          int         `json:"affver,omitempty"`
		Searchable      bool        `json:"searchable"`
		Hidden          bool        `json:"hidden"`
		Locked          bool        `json:"locked"`
		ValueType       string      `json:"valueType"`
		URI             string      `json:"uri"`
		Resizeable      bool        `json:"resizeable"`
		SortOrder       int         `json:"sortOrder"`
		Filterable      bool        `json:"filterable"`
		Label           string      `json:"label"`
		Sortable        bool        `json:"sortable"`
		SortAscending   bool        `json:"sortAscending"`
		ID              string      `json:"id"`
		Description     interface{} `json:"description"`
		LineWrap        bool        `json:"lineWrap"`
		PrimaryKeyOrder int         `json:"primaryKeyOrder,omitempty"`
		PrimaryKey      bool        `json:"primaryKey,omitempty"`
	} `json:"items"`
	FilteredRows int    `json:"filteredRows"`
	Identifier   string `json:"identifier"`
	TotalRows    int    `json:"totalRows"`
	NumRows      int    `json:"numRows"`
}

//Items struct used by Collect method
type Items struct {
	Items []struct {
		Properties []struct {
			ValueState   string      `json:"valueState"`
			Value        json.Number `json:"value"`
			Label        string      `json:"label"`
			ValueType    string      `json:"valueType"`
			ID           string      `json:"id"`
			DisplayValue string      `json:"displayValue"`
		} `json:"properties"`
		Tooltip     string `json:"tooltip"`
		Label       string `json:"label"`
		TypeLabel   string `json:"typeLabel"`
		Type        string `json:"type"`
		ID          string `json:"id"`
		Description string `json:"description"`
		URI         string `json:"uri"`
	} `json:"items"`
	FilteredRows int    `json:"filteredRows"`
	Identifier   string `json:"identifier"`
	TotalRows    int    `json:"totalRows"`
	NumRows      int    `json:"numRows"`
}

// Config struct that maps config yaml
type Config struct {
	ItmServerURL      string        `yaml:"itm_server_url"`
	ItmServerUser     string        `yaml:"itm_server_user"`
	ItmServerPassword string        `yaml:"itm_server_password"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	CollectionTimeout time.Duration `yaml:"collection_timeout"`
	Groups            []Groups      `yaml:"groups" validate:"required"`
}

// Groups struct is a subset of Config struct
type Groups struct {
	Name               string   `yaml:"name" validate:"required"`
	DatasetsURI        string   `yaml:"datasets_uri" validate:"required"`
	Labels             []string `yaml:"labels" validate:"required"`
	Metrics            []string `yaml:"metrics" validate:"required"`
	ManagedSystemGroup string   `yaml:"managed_system_group" validate:"required"`
}

// Result struct used by MakeAsyncRequest function it returns both response body and Attribute Group name to Collector method
type Result struct {
	group string
	body  []byte
}
