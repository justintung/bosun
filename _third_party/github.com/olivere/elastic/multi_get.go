// Copyright 2014 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type MultiGetService struct {
	client     *Client
	preference string
	realtime   *bool
	refresh    *bool
	items      []*MultiGetItem
}

func NewMultiGetService(client *Client) *MultiGetService {
	builder := &MultiGetService{
		client: client,
		items:  make([]*MultiGetItem, 0),
	}
	return builder
}

func (b *MultiGetService) Preference(preference string) *MultiGetService {
	b.preference = preference
	return b
}

func (b *MultiGetService) Refresh(refresh bool) *MultiGetService {
	b.refresh = &refresh
	return b
}

func (b *MultiGetService) Realtime(realtime bool) *MultiGetService {
	b.realtime = &realtime
	return b
}

func (b *MultiGetService) Add(items ...*MultiGetItem) *MultiGetService {
	b.items = append(b.items, items...)
	return b
}

func (b *MultiGetService) Source() interface{} {
	source := make(map[string]interface{})
	items := make([]interface{}, len(b.items))
	for i, item := range b.items {
		items[i] = item.Source()
	}
	source["docs"] = items
	return source
}

func (b *MultiGetService) Do() (*MultiGetResult, error) {
	// Build url
	urls := "/_mget"

	params := make(url.Values)
	if b.realtime != nil {
		params.Add("realtime", fmt.Sprintf("%v", *b.realtime))
	}
	if b.preference != "" {
		params.Add("preference", b.preference)
	}
	if b.refresh != nil {
		params.Add("refresh", fmt.Sprintf("%v", *b.refresh))
	}
	if len(params) > 0 {
		urls += "?" + params.Encode()
	}

	// Set up a new request
	req, err := b.client.NewRequest("GET", urls)
	if err != nil {
		return nil, err
	}

	// Set body
	req.SetBodyJson(b.Source())

	// Get response
	res, err := b.client.c.Do((*http.Request)(req))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	defer res.Body.Close()
	ret := new(MultiGetResult)
	if err := json.NewDecoder(res.Body).Decode(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// -- Multi Get Item --

// MultiGetItem is a single document to retrieve via the MultiGetService.
type MultiGetItem struct {
	index       string
	typ         string
	id          string
	routing     string
	fields      []string
	version     int64  // see org.elasticsearch.common.lucene.uid.Versions
	versionType string // see org.elasticsearch.index.VersionType
	fsc         *FetchSourceContext
}

func NewMultiGetItem() *MultiGetItem {
	return &MultiGetItem{
		version: -3, // MatchAny (see Version below)
	}
}

func (item *MultiGetItem) Index(index string) *MultiGetItem {
	item.index = index
	return item
}

func (item *MultiGetItem) Type(typ string) *MultiGetItem {
	item.typ = typ
	return item
}

func (item *MultiGetItem) Id(id string) *MultiGetItem {
	item.id = id
	return item
}

func (item *MultiGetItem) Routing(routing string) *MultiGetItem {
	item.routing = routing
	return item
}

func (item *MultiGetItem) Fields(fields ...string) *MultiGetItem {
	if item.fields == nil {
		item.fields = make([]string, 0)
	}
	item.fields = append(item.fields, fields...)
	return item
}

// Version can be MatchAny (-3), MatchAnyPre120 (0), NotFound (-1),
// or NotSet (-2). These are specified in org.elasticsearch.common.lucene.uid.Versions.
// The default is MatchAny (-3).
func (item *MultiGetItem) Version(version int64) *MultiGetItem {
	item.version = version
	return item
}

// VersionType can be "internal", "external", "external_gt", "external_gte",
// or "force". See org.elasticsearch.index.VersionType in Elasticsearch source.
// It is "internal" by default.
func (item *MultiGetItem) VersionType(versionType string) *MultiGetItem {
	item.versionType = versionType
	return item
}

func (item *MultiGetItem) FetchSource(fetchSourceContext *FetchSourceContext) *MultiGetItem {
	item.fsc = fetchSourceContext
	return item
}

// Source returns the serialized JSON to be sent to Elasticsearch as
// part of a MultiGet search.
func (item *MultiGetItem) Source() interface{} {
	source := make(map[string]interface{})

	if item.index != "" {
		source["_index"] = item.index
	}
	if item.typ != "" {
		source["_type"] = item.typ
	}
	source["_id"] = item.id

	if item.fsc != nil {
		source["_source"] = item.fsc.Source()
	}

	if item.fields != nil {
		source["_fields"] = item.fields
	}

	if item.routing != "" {
		source["_routing"] = item.routing
	}

	return source
}

// -- Result of a Multi Get request.

type MultiGetResult struct {
	Docs []*GetResult `json:"docs,omitempty"`
}
