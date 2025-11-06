package main

import (
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
)

type MQTTQuery struct {
	RefId string `json:"refId"`
	Hide  *bool  `json:"hide,omitempty"`

	Topic string `json:"topic"`
}

func (resource MQTTQuery) Equals(otherCandidate variants.Dataquery) bool {
	if otherCandidate == nil {
		return false
	}

	other, ok := otherCandidate.(MQTTQuery)
	if !ok {
		return false
	}

	if resource.RefId != other.RefId {
		return false
	}

	if resource.Hide == nil && other.Hide != nil || resource.Hide != nil && other.Hide == nil {
		return false
	}
	if resource.Hide != nil && *resource.Hide != *other.Hide {
		return false
	}

	return resource.Topic == other.Topic
}

func (resource MQTTQuery) Validate() error {
	return nil
}

// Let cog know that MQTTQuery is a Dataquery variant
func (resource MQTTQuery) ImplementsDataqueryVariant() {}

func (resource MQTTQuery) DataqueryType() string {
	return "custom"
}

func MQTTQueryVariantConfig() variants.DataqueryConfig {
	return variants.DataqueryConfig{
		Identifier: "mqtt-datasource", // datasource plugin ID
		DataqueryUnmarshaler: func(raw []byte) (variants.Dataquery, error) {
			dataquery := &MQTTQuery{}

			if err := json.Unmarshal(raw, dataquery); err != nil {
				return nil, err
			}

			return dataquery, nil
		},
	}
}

// Compile-time check to ensure that MQTTQuery indeed is
// a builder for variants.Dataquery
var _ cog.Builder[variants.Dataquery] = (*MQTTQueryBuilder)(nil)

type MQTTQueryBuilder struct {
	internal *MQTTQuery
}

func NewMQTTQueryBuilder(topic string) *MQTTQueryBuilder {
	return &MQTTQueryBuilder{
		internal: &MQTTQuery{Topic: topic},
	}
}

func (builder *MQTTQueryBuilder) Build() (variants.Dataquery, error) {
	if err := builder.internal.Validate(); err != nil {
		return MQTTQuery{}, err
	}

	return *builder.internal, nil
}

func (builder *MQTTQueryBuilder) RefId(refId string) *MQTTQueryBuilder {
	builder.internal.RefId = refId
	return builder
}

func (builder *MQTTQueryBuilder) Hide(hide bool) *MQTTQueryBuilder {
	builder.internal.Hide = &hide
	return builder
}
